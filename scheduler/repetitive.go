package scheduler

import (
	"fmt"
	"sync"
	"time"

	"context"

	"github.com/nyaxt/otaru/util"
	"go.uber.org/zap"
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

func (j *repetitiveJob) String() string {
	return fmt.Sprintf("repetitiveJob{id: %d, period: %v, task: %s}",
		j.id, j.period, util.Describe(j.task),
	)
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
	zap.S().Infof("RepetitiveJobRunner stop start")

	r.muJobs.Lock()
	defer r.muJobs.Unlock()
	for _, j := range r.jobs {
		j.mu.Lock()

		j.period = time.Duration(0)
		r.sched.Abort(j.scheduledJob)

		j.mu.Unlock()
	}
	zap.S().Infof("RepetitiveJobRunner stop done")
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

		if nextT.Before(now) {
			zap.S().Infof("repetitiveJob %v failed to execute within period %v. Scheduling next job to run after minimum period %v.", j, j.period, minimumPeriod)
			nextT = now.Add(minimumPeriod)
		}
	} else {
		nextT = now.Add(j.period)
	}

	j.scheduledJob = s.RunAt(j.task, nextT, func(v *JobView) { j.scheduleNext(s, v) })
	j.lastScheduledAt = nextT
}

type RepetitiveJobView struct {
	ID `json:"id"`

	TaskDesc string `json:"task_desc"`

	CreatedAt       time.Time `json:"created_at"`
	LastScheduledAt time.Time `json:"last_scheduled_at"`

	Period time.Duration `json:"period"`

	ScheduledJob ID `json:"scheduled_job"`
}

func (j *repetitiveJob) View() *RepetitiveJobView {
	j.mu.Lock()
	defer j.mu.Unlock()

	return &RepetitiveJobView{
		ID:              j.id,
		TaskDesc:        util.Describe(j.task),
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
		zap.S().Panicf("repetitiveJob ID %v is already taken. received duplicate: %v", j.id, j)
	}
	r.jobs[j.id] = j

	j.scheduleNext(r.sched, nil)

	return j.id
}

type SyncTask struct {
	S util.Syncer
}

func (t SyncTask) Run(ctx context.Context) Result {
	err := t.S.Sync()
	return ErrorResult{err}
}

func (t SyncTask) String() string {
	return fmt.Sprintf("SyncTask{%s}", util.TryGetImplName(t.S))
}

func (r *RepetitiveJobRunner) SyncEveryPeriod(s util.Syncer, period time.Duration) ID {
	return r.RunEveryPeriod(SyncTask{s}, period)
}

func (r *RepetitiveJobRunner) QueryAll() []*RepetitiveJobView {
	r.muJobs.Lock()
	defer r.muJobs.Unlock()

	ret := make([]*RepetitiveJobView, 0, len(r.jobs))
	for _, j := range r.jobs {
		ret = append(ret, j.View())
	}
	return ret
}

func (r *RepetitiveJobRunner) Query(id ID) *RepetitiveJobView {
	r.muJobs.Lock()
	defer r.muJobs.Unlock()

	j, ok := r.jobs[id]
	if !ok {
		return nil
	}
	return j.View()
}

func (r *RepetitiveJobRunner) abortWithLock(id ID) error {
	j, ok := r.jobs[id]
	if !ok {
		return fmt.Errorf("repetitiveJob ID %d not found.", id)
	}

	j.period = time.Duration(0)
	r.sched.Abort(j.scheduledJob)

	return nil
}

func (r *RepetitiveJobRunner) Abort(id ID) error {
	r.muJobs.Lock()
	defer r.muJobs.Unlock()

	return r.abortWithLock(id)
}
