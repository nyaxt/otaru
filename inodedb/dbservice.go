package inodedb

import (
	"log"
)

type DBTransactionRequest struct {
	tx      DBTransaction
	resultC chan error
}

type DBQueryNodeRequest struct {
	id      ID
	resultC chan NodeView
}

// DBService serializes requests to DBHandler
type DBService struct {
	txC     chan *DBTransactionRequest
	queryC  chan *DBQueryNodeRequest
	quitC   chan struct{}
	exitedC chan struct{}

	h DBHandler
}

var _ = DBHandler(&DBService{})

func NewDBService(h DBHandler) *DBService {
	s := &DBService{
		txC:     make(chan *DBTransactionRequest),
		queryC:  make(chan *DBQueryNodeRequest),
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
		case req := <-srv.txC:
			req.resultC <- srv.h.ApplyTransaction(req.tx)
		case req := <-srv.queryC:
			v, err := srv.h.QueryNode(req.id)
			if err != nil && !IsErrNotFound(err) {
				log.Printf("QueryNode failed and err isn't NotFound err: %v", err)
			}
			req.resultC <- v
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

func (srv *DBService) ApplyTransaction(tx DBTransaction) error {
	req := &DBTransactionRequest{tx: tx, resultC: make(chan error)}
	srv.txC <- req
	return <-req.resultC
}

func (srv *DBService) QueryNode(id ID) (NodeView, error) {
	req := &DBQueryNodeRequest{id: id, resultC: make(chan NodeView)}
	srv.queryC <- req
	v := <-req.resultC
	if v == nil {
		return nil, ErrNotFound
	}
	return v, nil
}
