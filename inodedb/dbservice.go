package inodedb

import (
	"fmt"

	"github.com/nyaxt/otaru/util"
)

// DBService serializes requests to DBHandler
type DBService struct {
	reqC    chan func()
	quitC   chan struct{}
	exitedC chan struct{}

	h DBHandler
}

var _ = DBHandler(&DBService{})
var _ = util.Syncer(&DBService{})

func NewDBService(h DBHandler) *DBService {
	s := &DBService{
		reqC:    make(chan func()),
		quitC:   make(chan struct{}),
		exitedC: make(chan struct{}),
		h:       h,
	}

	go s.run()

	return s
}

func (srv *DBService) run() {
	for {
		select {
		case f := <-srv.reqC:
			f()
		case <-srv.quitC:
			// FIXME: ensure that no req is pending
			close(srv.exitedC)
			return
		}
	}
}

func (srv *DBService) Quit() {
	srv.quitC <- struct{}{}
	<-srv.exitedC
}

func (srv *DBService) ApplyTransaction(tx DBTransaction) (txid TxID, err error) {
	ch := make(chan struct{})
	srv.reqC <- func() {
		txid, err = srv.h.ApplyTransaction(tx)
		close(ch)
	}
	<-ch
	return
}

func (srv *DBService) QueryNode(id ID, tryLock bool) (v NodeView, nlock NodeLock, err error) {
	ch := make(chan struct{})
	srv.reqC <- func() {
		v, nlock, err = srv.h.QueryNode(id, tryLock)
		close(ch)
	}
	<-ch
	return
}

func (srv *DBService) LockNode(id ID) (nlock NodeLock, err error) {
	ch := make(chan struct{})
	srv.reqC <- func() {
		nlock, err = srv.h.LockNode(id)
		close(ch)
	}
	<-ch
	return
}

func (srv *DBService) UnlockNode(nlock NodeLock) (err error) {
	ch := make(chan struct{})
	srv.reqC <- func() {
		err = srv.h.UnlockNode(nlock)
		close(ch)
	}
	<-ch
	return
}

func (srv *DBService) Sync() (err error) {
	ch := make(chan struct{})
	srv.reqC <- func() {
		if s, ok := srv.h.(util.Syncer); ok {
			err = s.Sync()
		}
		close(ch)
	}
	<-ch
	return
}

func (srv *DBService) GetStats() (stats DBServiceStats) {
	ch := make(chan struct{})
	srv.reqC <- func() {
		if prov, ok := srv.h.(DBServiceStatsProvider); ok {
			stats = prov.GetStats()
		}
		close(ch)
	}
	<-ch
	return
}

func (srv *DBService) QueryRecentTransactions() (txs []DBTransaction, err error) {
	ch := make(chan struct{})
	srv.reqC <- func() {
		if prov, ok := srv.h.(QueryRecentTransactionsProvider); ok {
			txs, err = prov.QueryRecentTransactions()
		} else {
			err = fmt.Errorf("DBHandler doesn't support QueryRecentTransactions")
		}
		close(ch)
	}
	<-ch
	return
}

func (srv *DBService) Fsck() (foundblobpaths []string, errs []error) {
	ch := make(chan struct{})
	srv.reqC <- func() {
		if prov, ok := srv.h.(DBFscker); ok {
			foundblobpaths, errs = prov.Fsck()
		} else {
			foundblobpaths, errs = nil, []error{fmt.Errorf("DBHandler doesn't support Fsck")}
		}
		close(ch)
	}
	<-ch
	return
}

func (*DBService) ImplName() string { return "inodedb.DBService" }
