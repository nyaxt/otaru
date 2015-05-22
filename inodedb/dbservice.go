package inodedb

import (
	"log"
)

type DBTransactionRequest struct {
	tx      DBTransaction
	resultC chan interface{}
}

type DBQueryNodeRequest struct {
	id      ID
	resultC chan NodeView
}

type DBLockNodeRequest struct {
	id      ID
	resultC chan interface{}
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
				v, err := srv.h.QueryNode(req.id)
				if err != nil && !IsErrNotFound(err) {
					log.Printf("QueryNode failed and err isn't NotFound err: %v", err)
				}
				req.resultC <- v
			case *DBLockNodeRequest:
				req := req.(*DBLockNodeRequest)
				nlock, err := srv.h.LockNode(req.id)
				if err != nil {
					req.resultC <- err
				} else {
					req.resultC <- nlock
				}
			default:
				log.Printf("unknown request passed to DBService: %v", req)
			}
		case <-srv.quitC:
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

func (srv *DBService) QueryNode(id ID) (NodeView, error) {
	req := &DBQueryNodeRequest{id: id, resultC: make(chan NodeView)}
	srv.reqC <- req
	v := <-req.resultC
	if v == nil {
		return nil, ENOENT
	}
	return v, nil
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
