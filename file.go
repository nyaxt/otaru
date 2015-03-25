package otaru

type FileChunk struct {
	Offset   int
	Length   int
	BlobPath string
}

type INodeID uint32

type INodeType int
const (
  FileNode = INodeType iota
  // DirectoryNode
  // SymlinkNode
)

type INode interface {
  ID() INodeID
  Type() INodeType
}

type INodeCommon struct {
  INodeID
  INodeType
}

func (n INodeCommon) ID() INodeId {
  return n.INodeID
}

func (n INodeCommon) Type() INodeType {
  return n.INodeType
}

type File struct {
  INodeCommon

	// OrigPath contains filepath passed to first create and does not necessary follow "rename" operations.
	// To be used for recovery/debug purposes only
	OrigPath string

	Chunks []FileChunk
}

func NewFile(id INodeID, origpath string) *File {
  return &File{
    INodeCommon{INodeID: id, INodeType: FileNode},
    OrigPath: origpath
  }
}

type INodeDB struct {
  nodes map[INodeID]INode
}

func (idb *INodeDB) Get(id INodeID) INode {
  return idb.nodes[id]
}

type FileSystem struct {
	INodeDB
  lastID INodeId
}

func (f *FileSystem) NewINodeID() INodeID {
  id := f.lastID + 1
  f.lastID = id
  return id
}

func (fs *FileSystem) CreateFile(otarupath string) (*File, error) {
  id := fs.NewINodeID()
  file := NewFile(otarupath)

  return file, nil
}

// func PRead(f File, offset int, p []byte) error
//   trivial
// func PWrite(f File, offset int, p []byte) error
//   ???
