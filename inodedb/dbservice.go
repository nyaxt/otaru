package inodedb

import (
	"log"

	"github.com/nyaxt/otaru/util"
)

type DBTransactionRequest struct {
	tx      DBTransaction
	resultC chan interface{}
}

type queryNodeResult struct {
	v     NodeView
	nlock NodeLock
	err   error
}

type DBQueryNodeRequest struct {
	id      ID
	tryLock bool
	resultC chan queryNodeResult
}

type DBLockNodeRequest struct {
	id      ID
	resultC chan interface{}
}

type DBUnlockNodeRequest struct {
	nlock   NodeLock
	resultC chan error
}

type DBSyncRequest struct {
	resultC chan error
}

type DBStatRequest struct {
	resultC chan DBServiceStats
}

// DBService serializes requests to DBHandler
type DBService struct {
	reqC    chan interface{}
	quitC   chan struct{}
	exitedC chan struct{}

	h DBHandler
}

var _ = DBHandler(&DBService{})
var _ = util.Syncer(&DBService{})

func NewDBService(h DBHandler) *DBService {
	s := &DBService{
		reqC:    make(chan interface{}),
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
		case req := <-srv.reqC:
			switch req.(type) {
			case *DBTransactionRequest:
				req := req.(*DBTransactionRequest)
				txid, err := srv.h.ApplyTransaction(req.tx)
				if err != nil {
					req.resultC <- err
				} else {
					req.resultC <- txid
				}
			case *DBQueryNodeRequest:
				req := req.(*DBQueryNodeRequest)
				v, nlock, err := srv.h.QueryNode(req.id, req.tryLock)
				req.resultC <- queryNodeResult{v, nlock, err}
			case *DBLockNodeRequest:
				req := req.(*DBLockNodeRequest)
				nlock, err := srv.h.LockNode(req.id)
				if err != nil {
					req.resultC <- err
				} else {
					req.resultC <- nlock
				}
			case *DBUnlockNodeRequest:
				req := req.(*DBUnlockNodeRequest)
				err := srv.h.UnlockNode(req.nlock)
				req.resultC <- err
			case *DBSyncRequest:
				req := req.(*DBSyncRequest)
				if s, ok := srv.h.(util.Syncer); ok {
					req.resultC <- s.Sync()
				} else {
					req.resultC <- nil
				}
			case *DBStatRequest:
				req := req.(*DBStatRequest)
				if prov, ok := srv.h.(DBServiceStatsProvider); ok {
					req.resultC <- prov.GetStats()
				} else {
					req.resultC <- DBServiceStats{}
				}
			default:
				log.Printf("unknown request passed to DBService: %v", req)
			}
		case <-srv.quitC:
			// FIXME: ensure that no req is pending
			srv.exitedC <- struct{}{}
			return
		}
	}
}

func (srv *DBService) Quit() {
	srv.quitC <- struct{}{}
	<-srv.exitedC
}

func (srv *DBService) ApplyTransaction(tx DBTransaction) (TxID, error) {
	req := &DBTransactionRequest{tx: tx, resultC: make(chan interface{})}
	srv.reqC <- req
	res := <-req.resultC
	if txid, ok := res.(TxID); ok {
		return txid, nil
	}
	return 0, res.(error)
}

func (srv *DBService) QueryNode(id ID, tryLock bool) (NodeView, NodeLock, error) {
	req := &DBQueryNodeRequest{id: id, tryLock: tryLock, resultC: make(chan queryNodeResult)}
	srv.reqC <- req
	res := <-req.resultC
	return res.v, res.nlock, res.err
}

func (srv *DBService) LockNode(id ID) (NodeLock, error) {
	req := &DBLockNodeRequest{id: id, resultC: make(chan interface{})}
	srv.reqC <- req
	res := <-req.resultC
	if nlock, ok := res.(NodeLock); ok {
		return nlock, nil
	}
	return NodeLock{}, res.(error)
}

func (srv *DBService) UnlockNode(nlock NodeLock) error {
	req := &DBUnlockNodeRequest{nlock: nlock, resultC: make(chan error)}
	srv.reqC <- req
	return <-req.resultC
}

func (srv *DBService) Sync() error {
	req := &DBSyncRequest{resultC: make(chan error)}
	srv.reqC <- req
	return <-req.resultC
}

func (srv *DBService) GetStats() DBServiceStats {
	req := &DBStatRequest{resultC: make(chan DBServiceStats)}
	srv.reqC <- req
	return <-req.resultC
}
