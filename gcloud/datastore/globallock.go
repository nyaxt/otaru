package datastore

import (
	"errors"
	"fmt"
	"math"
	"time"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"

	gcutil "github.com/nyaxt/otaru/gcloud/util"
	"github.com/nyaxt/otaru/logger"
)

var lklog = logger.Registry().Category("globallock")

const kindGlobalLock = "OtaruGlobalLock"

type lockEntry struct {
	CreatedAt time.Time `datastore:,noindex`
	HostName  string    `datastore:,noindex`
	Info      string    `datastore:,noindex`
}

type GlobalLocker struct {
	cfg          *Config
	rootKey      *datastore.Key
	lockEntryKey *datastore.Key

	lockEntry
}

func NewGlobalLocker(cfg *Config, hostname string, info string) *GlobalLocker {
	l := &GlobalLocker{
		cfg:     cfg,
		rootKey: datastore.NameKey(kindGlobalLock, cfg.rootKeyStr, nil),
		lockEntry: lockEntry{
			HostName: hostname,
			Info:     info,
		},
	}
	l.lockEntryKey = datastore.IDKey(kindGlobalLock, 1, l.rootKey)
	return l
}

type ErrLockTaken struct {
	CreatedAt time.Time
	HostName  string
	Info      string
}

var _ = error(&ErrLockTaken{})

func (e *ErrLockTaken) Error() string {
	return fmt.Sprintf("GlobalLock is taken by host \"%s\" at %s. Info: %s", e.HostName, e.CreatedAt, e.Info)
}

// Lock attempts to acquire the global lock.
// If the lock was already taken by other GlobalLocker instance, it will return an ErrLockTaken.
func (l *GlobalLocker) tryLockOnce() error {
	start := time.Now()
	cli, err := l.cfg.getClient(context.Background())
	if err != nil {
		return err
	}
	dstx, err := cli.NewTransaction(context.Background())
	if err != nil {
		return err
	}

	var e lockEntry
	if err := dstx.Get(l.lockEntryKey, &e); err != datastore.ErrNoSuchEntity {
		dstx.Rollback()
		if err == nil {
			return &ErrLockTaken{CreatedAt: e.CreatedAt, HostName: e.HostName, Info: e.Info}
		} else {
			return err
		}
	}

	l.lockEntry.CreatedAt = start
	if _, err := dstx.Put(l.lockEntryKey, &l.lockEntry); err != nil {
		dstx.Rollback()
		return err
	}
	if _, err := dstx.Commit(); err != nil {
		return err
	}

	logger.Infof(lklog, "GlobalLocker.tryLockOnce(%+v) took %s.", l.lockEntry, time.Since(start))
	return nil
}

func (l *GlobalLocker) Lock() (err error) {
	logger.Infof(lklog, "GlobalLocker.Lock() started.")
	return gcutil.RetryIfNeeded(func() error {
		return l.tryLockOnce()
	}, lklog)
}

// ForceUnlock releases the global lock entry forcibly, even if it was held by other GlobalLocker instance.
// If there was no lock, ForceUnlock will log an warning, but return no error.
func (l *GlobalLocker) forceUnlockOnce() error {
	start := time.Now()
	cli, err := l.cfg.getClient(context.Background())
	if err != nil {
		return err
	}
	dstx, err := cli.NewTransaction(context.Background())
	if err != nil {
		return err
	}

	var e lockEntry
	if err := dstx.Get(l.lockEntryKey, &e); err != nil {
		logger.Warningf(lklog, "GlobalLocker.ForceUnlock(): Force unlocking existing lock entry: %+v", e)
	}
	if err := dstx.Delete(l.lockEntryKey); err != nil {
		dstx.Rollback()
		if err == datastore.ErrNoSuchEntity {
			logger.Warningf(lklog, "GlobalLocker.ForceUnlock(): Warning: There was no global lock taken.")
			return nil
		}
		return err
	}

	if _, err := dstx.Commit(); err != nil {
		return err
	}

	logger.Infof(lklog, "GlobalLocker.forceUnlockOnce() took %s.", time.Since(start))
	return nil
}

func (l *GlobalLocker) ForceUnlock() error {
	logger.Infof(lklog, "GlobalLocker.ForceUnlock() started.")
	return gcutil.RetryIfNeeded(func() error {
		return l.forceUnlockOnce()
	}, lklog)
}

var ErrNoLock = errors.New("Attempted unlock, but couldn't find any lock entry.")

func closeEnough(a, b time.Time) bool {
	return math.Abs(a.Sub(b).Seconds()) < 2
}

const (
	checkCreatedAt  = false
	ignoreCreatedAt = true
)

// Unlock releases the global lock previously taken by this GlobalLocker.
// If the lock was taken by other GlobalLocker, Unlock will fail with ErrLockTaken.
// If there was no lock, Unlock will fail with ErrNoLock.
func (l *GlobalLocker) Unlock() error {
	logger.Infof(lklog, "GlobalLocker.Unlock() started.")
	return l.unlockInternal(checkCreatedAt)
}

func (l *GlobalLocker) UnlockIgnoreCreatedAt() error {
	logger.Infof(lklog, "GlobalLocker.UnlockIgnoreCreatedAt() started.")
	return l.unlockInternal(ignoreCreatedAt)
}

func checkLock(a, b lockEntry, checkCreatedAtFlag bool) bool {
	if checkCreatedAt && !closeEnough(a.CreatedAt, b.CreatedAt) {
		return false
	}
	if a.HostName != b.HostName {
		return false
	}
	return true
}

func (l *GlobalLocker) unlockInternalOnce(checkCreatedAtFlag bool) error {
	start := time.Now()

	cli, err := l.cfg.getClient(context.Background())
	if err != nil {
		return err
	}
	dstx, err := cli.NewTransaction(context.Background())
	if err != nil {
		return err
	}

	var e lockEntry
	if err := dstx.Get(l.lockEntryKey, &e); err != nil {
		dstx.Rollback()
		if err == datastore.ErrNoSuchEntity {
			return ErrNoLock
		} else {
			return err
		}
	}
	if !checkLock(l.lockEntry, e, checkCreatedAtFlag) {
		dstx.Rollback()
		return &ErrLockTaken{CreatedAt: e.CreatedAt, HostName: e.HostName, Info: e.Info}
	}
	if err := dstx.Delete(l.lockEntryKey); err != nil {
		dstx.Rollback()
		return err
	}

	if _, err := dstx.Commit(); err != nil {
		return err
	}

	logger.Infof(lklog, "GlobalLocker.unlockInternalOnce(%+v) took %s.", l.lockEntry, time.Since(start))
	return nil
}

func (l *GlobalLocker) unlockInternal(checkCreatedAtFlag bool) error {
	return gcutil.RetryIfNeeded(func() error {
		return l.unlockInternalOnce(checkCreatedAtFlag)
	}, lklog)
}

func (l *GlobalLocker) tryQueryOnce() (lockEntry, error) {
	var e lockEntry
	start := time.Now()

	cli, err := l.cfg.getClient(context.Background())
	if err != nil {
		return e, err
	}
	dstx, err := cli.NewTransaction(context.Background())
	if err != nil {
		return e, err
	}
	defer dstx.Rollback()

	if err := dstx.Get(l.lockEntryKey, &e); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return e, ErrNoLock
		} else {
			return e, err
		}
	}

	logger.Infof(lklog, "GlobalLocker.tryQueryOnce() took %s.", time.Since(start))
	return e, nil
}

func (l *GlobalLocker) Query() (le lockEntry, err error) {
	logger.Infof(lklog, "GlobalLocker.Query() started.")
	err = gcutil.RetryIfNeeded(func() error {
		le, err = l.tryQueryOnce()
		return err
	}, lklog)
	return
}
