package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/nyaxt/otaru/logger"
)

type repetitiveJob struct {
	id ID

	createdAt       time.Time
	lastScheduledAt time.Time

	task   Task
	period time.Duration

	scheduledJob ID

	mu sync.Mutex
}

type RepetitiveJobRunner struct {
	sched *Scheduler

	idGen

	muJobs sync.Mutex
	jobs   map[ID]*repetitiveJob
}

func NewRepetitiveJobRunner(sched *Scheduler) *RepetitiveJobRunner {
	r := &RepetitiveJobRunner{
		sched: sched,
		idGen: idGen{0},
		jobs:  make(map[ID]*repetitiveJob, 0),
	}

	return r
}

func (r *RepetitiveJobRunner) Stop() {
	logger.Infof(mylog, "RepetitiveJobRunner stop start")

	r.muJobs.Lock()
	defer r.muJobs.Unlock()
	for _, j := range r.jobs {
		j.mu.Lock()

		// FIXME ABORT!!!

		j.mu.Unlock()
	}
	logger.Infof(mylog, "RepetitiveJobRunner stop done")
}

var minimumPeriod time.Duration = 300 * time.Millisecond

func (j *repetitiveJob) scheduleNext(s *Scheduler, v *JobView) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.period.Nanoseconds() == 0 {
		return
	}

	now := time.Now()

	var nextT time.Time
	if v != nil {
		nextT = j.lastScheduledAt.Add(j.period)
		logger.Debugf(mylog, "now: %v, nextT: %v", now, nextT)

		if nextT.Before(now) {
			logger.Infof(mylog, "repetitiveJob %v failed to execute within period %v. Scheduling next job to run after minimum period %v.", j, j.period, minimumPeriod)
			nextT = now.Add(minimumPeriod)
		}
	} else {
		nextT = now.Add(j.period)
	}

	j.scheduledJob = s.RunAt(j.task, nextT, func(v *JobView) { j.scheduleNext(s, v) })
	j.lastScheduledAt = now
}

type RepetitiveJobView struct {
	ID `json:"id"`

	CreatedAt       time.Time `json:"created_at"`
	LastScheduledAt time.Time `json:"last_scheduled_at"`

	Period time.Duration `json:"period"`

	ScheduledJob ID `json:"scheduled_job"`
}

func (j *repetitiveJob) View() RepetitiveJobView {
	j.mu.Lock()
	defer j.mu.Unlock()

	return RepetitiveJobView{
		ID:              j.id,
		CreatedAt:       j.createdAt,
		LastScheduledAt: j.lastScheduledAt,
		Period:          j.period,
		ScheduledJob:    j.scheduledJob,
	}
}

func (r *RepetitiveJobRunner) RunEveryPeriod(t Task, period time.Duration) ID {
	now := time.Now()
	j := &repetitiveJob{
		id: r.idGen.genID(),

		createdAt: now,

		task:   t,
		period: period,
	}

	r.muJobs.Lock()
	defer r.muJobs.Unlock()

	if _, ok := r.jobs[j.id]; ok {
		logger.Panicf(mylog, "repetitiveJob ID %v is already taken. received duplicate: %v", j.id, j)
	}
	r.jobs[j.id] = j

	j.scheduleNext(r.sched, nil)

	return j.id
}

func (r *RepetitiveJobRunner) Query(id ID) RepetitiveJobView {
	r.muJobs.Lock()
	defer r.muJobs.Unlock()

	j, ok := r.jobs[id]
	if !ok {
		return RepetitiveJobView{}
	}
	return j.View()
}

func (r *RepetitiveJobRunner) Abort(id ID) error {
	r.muJobs.Lock()
	defer r.muJobs.Unlock()

	j, ok := r.jobs[id]
	if !ok {
		return fmt.Errorf("repetitiveJob ID %d not found.", id)
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	j.period = time.Duration(0)
	r.sched.Abort(j.scheduledJob)

	return nil
}
