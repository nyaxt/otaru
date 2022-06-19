package scheduler

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"context"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
	"go.uber.org/zap"
)

var mylog = logger.Registry().Category("scheduler")

type Result interface {
	Err() error
}

type ErrorResult struct {
	Error error
}

func (er ErrorResult) Err() error    { return er.Error }
func (ErrorResult) ImplName() string { return "ErrorResult" }

func (er ErrorResult) MarshalJSON() ([]byte, error) {
	type jsonRep struct {
		ErrorString string `json:"error_str,omitempty"`
	}

	var errstr string
	if er.Error != nil {
		errstr = er.Error.Error()
	}

	return json.Marshal(jsonRep{ErrorString: errstr})
}

type Task interface {
	Run(ctx context.Context) Result
}

type State int32

const (
	JobScheduled = iota
	JobStarted   = iota
	JobFinished  = iota
	JobAborted   = iota
)

func (st State) String() string {
	switch st {
	case JobScheduled:
		return "JobScheduled"
	case JobStarted:
		return "JobStarted"
	case JobFinished:
		return "JobFinished"
	case JobAborted:
		return "JobAborted"
	default:
		return "UnknownJobState"
	}
}

var ZeroTime time.Time

type job struct {
	ID
	State
	// Issuer string

	CreatedAt   time.Time
	ScheduledAt time.Time
	StartedAt   time.Time
	FinishedAt  time.Time

	Task
	Result
	DoneCallback

	mu         sync.Mutex
	cancelfn   context.CancelFunc
	scheduledC chan struct{}
}

func (j *job) String() string {
	return fmt.Sprintf("job{ID: %d, CreatedAt: %v, Task: %s, State: %v}",
		j.ID,
		j.CreatedAt,
		util.Describe(j.Task),
		j.State,
	)
}

type JobView struct {
	ID    `json:"id"`
	State `json:"state"`
	// Issuer string

	TaskDesc string `json:"task_desc"`

	CreatedAt   time.Time `json:"created_at"`
	ScheduledAt time.Time `json:"scheduled_at"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`

	Result `json:"result"`
}

type DoneCallback func(*JobView)

func (j *job) ViewWithLock() *JobView {
	return &JobView{
		ID:          j.ID,
		State:       j.State,
		TaskDesc:    util.Describe(j.Task),
		CreatedAt:   j.CreatedAt,
		ScheduledAt: j.ScheduledAt,
		StartedAt:   j.StartedAt,
		FinishedAt:  j.FinishedAt,
		Result:      j.Result,
	}
}

func (j *job) View() *JobView {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.ViewWithLock()
}

type jobQuery struct {
	ID
	resultC chan []*JobView
}

type abortReq struct {
	ID
	doneC chan struct{}
}

type Scheduler struct {
	numRunners  int
	numWaitJobs int

	idGen

	scheduleC     chan *job
	queryC        chan *jobQuery
	abortC        chan *abortReq
	runC          chan *job
	forceGCC      chan struct{}
	joinScheduleC chan struct{}
	joinRunnerC   chan struct{}

	ZombiePeriod time.Duration
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		numRunners:    4, // FIXME
		numWaitJobs:   0,
		idGen:         idGen{0},
		scheduleC:     make(chan *job, 1),
		queryC:        make(chan *jobQuery, 1),
		abortC:        make(chan *abortReq, 1),
		runC:          make(chan *job, 8),
		forceGCC:      make(chan struct{}),
		joinScheduleC: make(chan struct{}),
		joinRunnerC:   make(chan struct{}),

		ZombiePeriod: 30 * time.Second,
	}

	go s.schedulerMain()
	for i := 0; i < s.numRunners; i++ {
		go s.runnerMain()
	}

	return s
}

type Stats struct {
	NumRunners  int `json:"num_runners"`
	NumWaitJobs int `json:"num_wait_jobs"`
}

func (s *Scheduler) GetStats() *Stats {
	return &Stats{
		NumRunners:  s.numRunners,
		NumWaitJobs: s.numWaitJobs,
	}
}

func (s *Scheduler) RunAt(task Task, at time.Time, cb DoneCallback) ID {
	id := s.idGen.genID()
	j := &job{
		ID:           id,
		CreatedAt:    time.Now(),
		ScheduledAt:  at,
		Task:         task,
		DoneCallback: cb,
		scheduledC:   make(chan struct{}),
	}

	s.scheduleC <- j
	<-j.scheduledC

	return id
}

func (s *Scheduler) RunImmediately(task Task, cb DoneCallback) ID {
	return s.RunAt(task, ZeroTime, cb)
}

func (s *Scheduler) RunImmediatelyBlock(task Task) *JobView {
	doneC := make(chan *JobView)
	s.RunAt(task, ZeroTime, func(v *JobView) { doneC <- v; close(doneC) })
	return <-doneC
}

func (s *Scheduler) Query(id ID) *JobView {
	q := &jobQuery{
		ID:      id,
		resultC: make(chan []*JobView),
	}
	s.queryC <- q
	rs := <-q.resultC
	if rs == nil {
		return nil
	}
	return rs[0]
}

func (s *Scheduler) QueryAll() []*JobView {
	q := &jobQuery{
		ID:      allJobs,
		resultC: make(chan []*JobView),
	}
	s.queryC <- q
	return <-q.resultC
}

func (s *Scheduler) abortInternal(id ID) {
	req := &abortReq{
		ID:    id,
		doneC: make(chan struct{}),
	}
	s.abortC <- req
	<-req.doneC
}

func (s *Scheduler) Abort(id ID) {
	if id == allJobs {
		// allJobs should only be used internally
		return
	}

	s.abortInternal(id)
}

func (s *Scheduler) stop() {
	zap.S().Infof("scheduler stop start")
	close(s.scheduleC)
	close(s.queryC)
	close(s.abortC)

	<-s.joinScheduleC
	for i := 0; i < s.numRunners; i++ {
		<-s.joinRunnerC
	}
	close(s.forceGCC)
	zap.S().Infof("scheduler stop done")
}

func (s *Scheduler) RunAllAndStop() { s.stop() }

func (s *Scheduler) AbortAllAndStop() {
	s.abortInternal(allJobs)
	s.stop()
}

func abortJob(j *job) {
	j.mu.Lock()
	switch j.State {
	case JobScheduled:
		j.State = JobAborted
		if j.DoneCallback != nil {
			go j.DoneCallback(j.ViewWithLock())
		}
	case JobStarted:
		j.cancelfn()
	case JobFinished:
		// Job has already finished. Too late.
	case JobAborted:
		// Job is already aborted. Nothing to do.
	}
	j.mu.Unlock()
}

func (s *Scheduler) ForceGC() { s.forceGCC <- struct{}{} }

const gcPeriod = 10 * time.Second

func (s *Scheduler) schedulerMain() {
	var tickC <-chan time.Time
	tickC = make(chan time.Time)
	var nextWakeUp time.Time
	var timer *time.Timer
	renewTimer := func() {
		d := nextWakeUp.Sub(time.Now())
		if timer == nil {
			timer = time.NewTimer(d)
		} else {
			timer.Stop()
			timer.Reset(d)
		}
		tickC = timer.C
	}

	waitJobs := make([]*job, 0)
	jobs := make(map[ID]*job)

	gcTickC := time.Tick(gcPeriod)
	doGC := func() {
		zap.S().Debugf("JobGC start.")
		gcstart := time.Now()
		threshold := gcstart.Add(-s.ZombiePeriod)

		oldJobIDs := make([]ID, 0)
		for id, j := range jobs {
			j.mu.Lock()
			if j.State == JobAborted || j.State == JobFinished {
				if j.FinishedAt.Before(threshold) {
					zap.S().Debugf("GC-ing job %v: %v since aborted/finished.", j, gcstart.Sub(j.FinishedAt))
					oldJobIDs = append(oldJobIDs, id)
				}
			}
			j.mu.Unlock()
		}
		for _, id := range oldJobIDs {
			delete(jobs, id)
		}

		zap.S().Debugf("JobGC end. Took %v. Deleted %d entries", time.Since(gcstart), len(oldJobIDs))
	}

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

				// kick scheduler
				nextWakeUp = time.Now()
				renewTimer()
				continue
			}

			if _, ok := jobs[j.ID]; ok {
				zap.S().Errorf("job ID %v is already taken. received duplicate: %v", j.ID, j)
				if j.scheduledC != nil {
					close(j.scheduledC)
				}
				continue
			}
			jobs[j.ID] = j

			now := time.Now()
			if j.ScheduledAt.Before(now) {
				s.runC <- j
			} else {
				if nextWakeUp.IsZero() || nextWakeUp.After(j.ScheduledAt) {
					nextWakeUp = j.ScheduledAt
					renewTimer()
				}

				waitJobs = append(waitJobs, j)
				s.numWaitJobs = len(waitJobs)
			}
			if j.scheduledC != nil {
				close(j.scheduledC)
			}

		case q := <-s.queryC:
			if q == nil {
				continue
			}
			if q.ID == allJobs {
				jvs := make([]*JobView, 0, len(jobs))
				for _, j := range jobs {
					jvs = append(jvs, j.View())
				}
				q.resultC <- jvs
				continue
			}

			id := q.ID
			j, ok := jobs[id]
			if !ok {
				q.resultC <- nil
			} else {
				q.resultC <- []*JobView{j.View()}
			}

		case req := <-s.abortC:
			if req == nil {
				continue
			}

			if req.ID == allJobs {
				for _, j := range jobs {
					abortJob(j)
				}

				req.doneC <- struct{}{}
				continue
			}

			j, ok := jobs[req.ID]
			if !ok {
				zap.S().Warnf("Abort target job ID %d doesn't exist.", req.ID)
				req.doneC <- struct{}{}
				continue
			}
			abortJob(j)
			req.doneC <- struct{}{}

		case <-tickC:
			stillWaitJobs := make([]*job, 0, len(waitJobs))
			now := time.Now()
			var zero time.Time
			nextWakeUp = zero
			for _, j := range waitJobs {
				if j.State != JobScheduled {
					continue
				}
				if j.ScheduledAt.Before(now) {
					s.runC <- j
				} else {
					if nextWakeUp.IsZero() || nextWakeUp.After(j.ScheduledAt) {
						nextWakeUp = j.ScheduledAt
					}

					stillWaitJobs = append(stillWaitJobs, j)
				}
			}
			waitJobs = stillWaitJobs
			s.numWaitJobs = len(waitJobs)

			renewTimer()

		case <-gcTickC:
			doGC()
		case <-s.forceGCC:
			doGC()
		}
	}
}

func (s *Scheduler) runnerMain() {
	for j := range s.runC {
		j.mu.Lock()
		if j.State == JobAborted {
			j.mu.Unlock()
			continue
		}
		if j.State != JobScheduled {
			zap.S().Infof("Skipping job not in scheduled state: %v", j)

			j.mu.Unlock()
			continue
		}

		task := j.Task
		ctx, cancelfn := context.WithCancel(context.Background())
		j.cancelfn = cancelfn
		j.StartedAt = time.Now()
		j.State = JobStarted

		j.mu.Unlock()
		zap.S().Debugf("About to run job %v", j)
		result := task.Run(ctx)
		if result != nil {
			if err := result.Err(); err != nil {
				zap.S().Warnf("Job %v failed with error: %v", j, err)
			}
		}
		finishedAt := time.Now()
		j.mu.Lock()

		j.Result = result
		j.FinishedAt = finishedAt
		if ctx.Err() != nil {
			j.State = JobAborted
		} else {
			j.State = JobFinished
		}
		if j.DoneCallback != nil {
			go j.DoneCallback(j.ViewWithLock())
		}
		j.mu.Unlock()
	}
	s.joinRunnerC <- struct{}{}
}
