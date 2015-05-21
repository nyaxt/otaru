package inodedb

type NodeView interface {
	GetID() ID
	GetType() Type
}

type FileNodeView interface {
	NodeView
	GetSize() int64
	GetChunks() []FileChunk
}

type DirNodeView interface {
	NodeView
	GetEntries() map[string]ID
}

type DBHandler interface {
	ApplyTransaction(tx DBTransaction) error
	QueryNode(id ID) (NodeView, error)
}

type DBServiceRequest struct {
	tx      DBTransaction
	resultC chan error
}

type DBService struct {
	c chan DBServiceRequest
	h DBHandler
}

func NewDBService(h DBHandler) *DBService {
	return &DBService{
		c: make(chan DBServiceRequest),
		h: h,
	}
}

/*

File write:
- Acquire lock when opened with write perm
{
  - get old chunks
  - cs <- add new chunk ** not cancellable **
  - save new cs
}
- keep renewing the lock
- release the lock when done

Rename:
atomic {
  - link new dir
  - unlink old dir
}

CreateFile:
atomic {
  - create new file node
  - link new dir
}

*/
