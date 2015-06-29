package scheduler

import (
	"time"
)

type Result interface {
	Err() error
}

type Task interface {
	Run() Result
}

type State int

const (
	JobScheduled = iota
	JobStarted   = iota
	JobFinished  = iota
)

func (st State) String() string {
	switch st {
	case JobScheduled:
		return "JobScheduled"
	case JobStarted:
		return "JobStarted"
	case JobFinished:
		return "JobFinished"
	default:
		return "Unknown"
	}
}

var ZeroTime time.Time

type Job struct {
	ID
	State
	// Issuer string

	CreatedAt   time.Time
	ScheduledAt time.Time
	StartedAt   time.Time
	FinishedAt  time.Time

	Task
	Result
}

type Scheduler struct {
	numRunners int

	idGen

	scheduleC     chan *Job
	runC          chan *Job
	joinScheduleC chan struct{}
	joinRunnerC   chan struct{}

	numWaitJobs int
}

const schedulerTickDuration = 300 * time.Millisecond

func NewScheduler() *Scheduler {
	s := &Scheduler{
		numRunners:    4, // FIXME
		idGen:         idGen{1},
		scheduleC:     make(chan *Job, 16),
		runC:          make(chan *Job, 1),
		joinScheduleC: make(chan struct{}),
		joinRunnerC:   make(chan struct{}),
	}

	go s.schedulerMain()
	for i := 0; i < s.numRunners; i++ {
		go s.runnerMain()
	}

	return s
}

func (s *Scheduler) RunAt(task Task, at time.Time) ID {
	id := s.idGen.genID()
	j := &Job{
		ID:          id,
		CreatedAt:   time.Now(),
		ScheduledAt: at,
		Task:        task,
	}

	s.scheduleC <- j

	return id
}

func (s *Scheduler) RunImmediately(task Task) ID {
	return s.RunAt(task, ZeroTime)
}

func (s *Scheduler) RunAllAndStop() {
	close(s.scheduleC)

	<-s.joinScheduleC
	for i := 0; i < s.numRunners; i++ {
		<-s.joinRunnerC
	}
}

func (s *Scheduler) schedulerMain() {
	tick := time.NewTicker(schedulerTickDuration) // FIXME: This should actually wait until next scheduled task instead of using fixed duration ticker.
	waitJobs := make([]*Job, 0)

	defer func() {
		close(s.runC)
		s.joinScheduleC <- struct{}{}
	}()

	scheduleC := s.scheduleC
	for {
		if scheduleC == nil && s.numWaitJobs == 0 {
			return
		}

		select {
		case j, more := <-scheduleC:
			if !more {
				// stop polling on scheduleC
				scheduleC = nil
				continue
			}
			if j.ScheduledAt.Before(time.Now()) {
				s.runC <- j
			} else {
				waitJobs = append(waitJobs, j)
				s.numWaitJobs = len(waitJobs)
			}
		case <-tick.C:
			stillWaitJobs := make([]*Job, 0, len(waitJobs))
			now := time.Now()
			for _, j := range waitJobs {
				if j.ScheduledAt.Before(now) {
					s.runC <- j
				} else {
					stillWaitJobs = append(stillWaitJobs, j)
				}
			}
			waitJobs = stillWaitJobs
			s.numWaitJobs = len(waitJobs)
		}
	}
}

func (s *Scheduler) runnerMain() {
	for j := range s.runC {
		j.StartedAt = time.Now()
		j.Result = j.Task.Run()
		j.FinishedAt = time.Now()
	}
	s.joinRunnerC <- struct{}{}
}
