package inodedb

import (
	"fmt"
)

type DBOperation interface {
	Apply(s *DBState) error
}

type OpMeta struct {
	Kind string `json:"kind"`
}

type InitializeFileSystemOp struct {
	OpMeta `json:",inline"`
}

var _ = DBOperation(&InitializeFileSystemOp{})

const RootDirID ID = 1

func (op *InitializeFileSystemOp) Apply(s *DBState) error {
	if len(s.nodes) != 0 {
		return fmt.Errorf("DB not empty. Already contains %d nodes!", len(s.nodes))
	}
	if s.lastID != 0 {
		return fmt.Errorf("DB lastId != 0")
	}

	n := &DirNode{
		INodeCommon: INodeCommon{ID: RootDirID, OrigPath: "/"},
		Entries:     make(map[string]ID),
	}

	if err := s.addNewNode(n); err != nil {
		return fmt.Errorf("Failed to create root DirNode: %v", err)
	}
	s.lastID = 1

	return nil
}

type CreateDirOp struct {
	OpMeta   `json:",inline"`
	NodeLock `json:"nodelock"`
	OrigPath string `json:"origpath"`
}

var _ = DBOperation(&CreateDirOp{})

func (op *CreateDirOp) Apply(s *DBState) error {
	if err := s.checkLock(op.NodeLock, true); err != nil {
		return err
	}

	n := &DirNode{
		INodeCommon: INodeCommon{ID: op.ID, OrigPath: op.OrigPath},
		Entries:     make(map[string]ID),
	}

	if err := s.addNewNode(n); err != nil {
		return fmt.Errorf("Failed to create new DirNode: %v", err)
	}

	return nil
}

type CreateFileOp struct {
	OpMeta   `json:",inline"`
	NodeLock `json:"nodelock"`
	OrigPath string `json:"origpath"`
}

func (op *CreateFileOp) Apply(s *DBState) error {
	if err := s.checkLock(op.NodeLock, true); err != nil {
		return err
	}

	n := &FileNode{
		INodeCommon: INodeCommon{ID: op.ID, OrigPath: op.OrigPath},
		Size:        0,
	}

	if err := s.addNewNode(n); err != nil {
		return fmt.Errorf("Failed to create new FileNode: %v", err)
	}

	return nil
}

type HardLinkOp struct {
	OpMeta   `json:",inline"`
	NodeLock `json:"nodelock"`
	Name     string `json:"name"`
	TargetID ID     `json:"targetid"`
}

func (op *HardLinkOp) Apply(s *DBState) error {
	if err := s.checkLock(op.NodeLock, false); err != nil {
		return err
	}

	n, ok := s.nodes[op.ID]
	if !ok {
		return ENOENT
	}
	dn, ok := n.(*DirNode)
	if !ok {
		return ENOTDIR
	}

	if _, ok := s.nodes[op.TargetID]; !ok {
		return ENOENT
	}

	if _, ok := dn.Entries[op.Name]; ok {
		return EEXIST
	}
	dn.Entries[op.Name] = op.TargetID

	return nil
}

type UpdateChunksOp struct {
	OpMeta   `json:",inline"`
	NodeLock `json:"nodelock"`
	Chunks   []FileChunk `json:"chunks"`
}

func (op *UpdateChunksOp) Apply(s *DBState) error {
	if err := s.checkLock(op.NodeLock, true); err != nil {
		return err
	}

	n, ok := s.nodes[op.ID]
	if !ok {
		return ENOENT
	}
	fn, ok := n.(*FileNode)
	if !ok {
		return fmt.Errorf("UpdateChunksOp specified node was not file node but was type: %d", n.GetType())
	}

	fn.Chunks = op.Chunks // FIXME: not sure if need clone?
	return nil
}

type UpdateSizeOp struct {
	OpMeta   `json:",inline"`
	NodeLock `json:"nodelock"`
	Size     int64 `json:"size"`
}

func (op *UpdateSizeOp) Apply(s *DBState) error {
	if err := s.checkLock(op.NodeLock, true); err != nil {
		return err
	}

	n, ok := s.nodes[op.ID]
	if !ok {
		return ENOENT
	}
	fn, ok := n.(*FileNode)
	if !ok {
		return fmt.Errorf("UpdateChunksOp specified node was not file node but was type: %d", n.GetType())
	}

	fn.Size = op.Size
	return nil
}

type RenameOp struct {
	OpMeta   `json:",inline"`
	SrcDirID ID     `json:"srcdir"`
	SrcName  string `json:"srcname"`
	DstDirID ID     `json:"dstdir"`
	DstName  string `json:"dstname"`
}

func (op *RenameOp) Apply(s *DBState) error {
	if err := s.checkLock(NodeLock{op.SrcDirID, NoTicket}, false); err != nil {
		return err
	}
	if err := s.checkLock(NodeLock{op.DstDirID, NoTicket}, false); err != nil {
		return err
	}

	srcn, ok := s.nodes[op.SrcDirID]
	if !ok {
		return ENOENT
	}
	dstn, ok := s.nodes[op.DstDirID]
	if !ok {
		return ENOENT
	}

	srcdn, ok := srcn.(*DirNode)
	if !ok {
		return ENOTDIR
	}
	dstdn, ok := dstn.(*DirNode)
	if !ok {
		return ENOTDIR
	}

	if _, ok = dstdn.Entries[op.DstName]; ok {
		return EEXIST
	}

	id, ok := srcdn.Entries[op.SrcName]
	if !ok {
		return ENOENT
	}

	delete(srcdn.Entries, op.SrcName)
	dstdn.Entries[op.DstName] = id
	return nil
}

type RemoveOp struct {
	OpMeta   `json:",inline"`
	NodeLock `json:"nodelock"`
	Name     string `json:"name"`
}

func (op *RemoveOp) Apply(s *DBState) error {
	if err := s.checkLock(op.NodeLock, false); err != nil {
		return err
	}

	n, ok := s.nodes[op.ID]
	if !ok {
		return ENOENT
	}
	dn, ok := n.(*DirNode)
	if !ok {
		return ENOTDIR
	}

	if _, ok := dn.Entries[op.Name]; !ok {
		return ENOENT
	}

	delete(dn.Entries, op.Name)
	return nil
}
