package scheduler

import (
	"time"
)

type TaskResult interface {
	Err() error
}

type Task interface {
	Run() TaskResult
}

type Job struct {
	ID int
	// Issuer string

	CreatedAt   time.Time
	ScheduledAt time.Time
	StartedAt   time.Time
	FinishedAt  time.Time

	Task
}

type Scheduler struct {
	nextJobID int
	scheduleC chan *Job
	runC      chan *Job
}

const schedulerTickDuration = 1 * time.Second

func NewScheduler() *Scheduler {
	s := &Scheduler{
		nextJobID: 1,
		scheduleC: make(chan *Job, 16),
	}

	go s.schedulerMain()

	return s
}

func (s *Scheduler) RunTaskAt(task Task, at time.Time) {
	j := &Job{
		ID:          s.nextJobID,
		CreatedAt:   time.Now(),
		ScheduledAt: at,
		Task:        task,
	}
	s.nextJobID++

	s.scheduleC <- s
}

func (s *Scheduler) schedulerMain() {
	tick := time.NewTicker(schedulerTickDuration) // FIXME: This should actually wait until next scheduled task instead of using fixed duration ticker.
	select {
	case j := <-s.scheduleC:
		now := time.Now()
		if j.ScheduledAt.Before(now) {
			runC <- j
		} else {

		}
	case <-tick.C:

	}
	close(runC)
}
