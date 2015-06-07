package inodedb

import (
	"log"
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

// DBService serializes requests to DBHandler
type DBService struct {
	reqC    chan interface{}
	quitC   chan struct{}
	exitedC chan struct{}

	h DBHandler
}

var _ = DBHandler(&DBService{})

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
