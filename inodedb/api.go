package inodedb

type NodeView interface {
	GetID() ID
	GetType() Type
}

type FileNodeView interface {
	NodeView
	GetChunks() []FileChunk
}

type DirNodeView interface {
	NodeView
	GetEntries() map[string]ID
}

type DBHandle interface {
	CreateFile(name string) (FileNodeView, error)
	UpdateFileChunks(id ID, cs []FileChunk) (FileNodeView, error)

	CreateDir(dirID ID, name string) (DirNodeView, error)
	AddDirEntry(dirID, id ID) (DirNodeView, error)
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
