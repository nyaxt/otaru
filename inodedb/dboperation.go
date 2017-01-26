package inodedb

import (
	"errors"
	"fmt"
	"time"

	"github.com/nyaxt/otaru/util"
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
		INodeCommon: INodeCommon{ID: RootDirID, OrigPath: "/", Uid: 0, Gid: 0, PermMode: 0777, ModifiedT: time.Now()},
		ParentID:    RootDirID,
		Entries:     make(map[string]ID),
	}

	if err := s.addNewNode(n); err != nil {
		return fmt.Errorf("Failed to create root DirNode: %v", err)
	}
	s.lastID = 1

	return nil
}

type CreateNodeOp struct {
	OpMeta   `json:",inline"`
	NodeLock `json:"nodelock"`
	OrigPath string `json:"origpath"`
	Type     `json:"type"`

	ParentID ID `json:"parent_id"` // only valid for DirNodeT

	Uid       uint32    `json:"uid"`
	Gid       uint32    `json:"gid"`
	PermMode  uint16    `json:"perm_mode"`
	ModifiedT time.Time `json:"modified_t"`
}

var _ = DBOperation(&CreateNodeOp{})

func (op *CreateNodeOp) Apply(s *DBState) error {
	if err := s.checkLock(op.NodeLock, true); err != nil {
		return err
	}

	var n INode
	switch op.Type {
	case FileNodeT:
		n = &FileNode{
			INodeCommon: INodeCommon{ID: op.ID, OrigPath: op.OrigPath, Uid: op.Uid, Gid: op.Gid, PermMode: op.PermMode, ModifiedT: time.Now()},
			Size:        0,
		}
	case DirNodeT:
		n = &DirNode{
			INodeCommon: INodeCommon{ID: op.ID, OrigPath: op.OrigPath, Uid: op.Uid, Gid: op.Gid, PermMode: op.PermMode, ModifiedT: time.Now()},
			ParentID:    op.ParentID,
			Entries:     make(map[string]ID),
		}
	default:
		return fmt.Errorf("Unknown node type specified to CreateNodeOp: %v", op.Type)
	}

	if err := s.addNewNode(n); err != nil {
		return fmt.Errorf("Failed to create new DirNode: %v", err)
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
		return util.ENOENT
	}
	dn, ok := n.(*DirNode)
	if !ok {
		return util.ENOTDIR
	}

	if _, ok := s.nodes[op.TargetID]; !ok {
		return util.ENOENT
	}

	if _, ok := dn.Entries[op.Name]; ok {
		return util.EEXIST
	}
	dn.Entries[op.Name] = op.TargetID
	dn.ModifiedT = time.Now()

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
		return util.ENOENT
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
		return util.ENOENT
	}
	fn, ok := n.(*FileNode)
	if !ok {
		return fmt.Errorf("UpdateChunksOp specified node was not file node but was type: %d", n.GetType())
	}

	fn.Size = op.Size
	fn.ModifiedT = time.Now()
	return nil
}

type UpdateUidOp struct {
	OpMeta `json:",inline"`
	ID     `json:"id"`
	Uid    uint32 `json:"uid"`
}

func (op *UpdateUidOp) Apply(s *DBState) error {
	n, ok := s.nodes[op.ID]
	if !ok {
		return util.ENOENT
	}
	switch n := n.(type) {
	case *FileNode:
		n.Uid = op.Uid
	case *DirNode:
		n.Uid = op.Uid
	default:
		return fmt.Errorf("UpdateUidOp: Unsupported node type: %d", n.GetType())
	}
	return nil
}

type UpdateGidOp struct {
	OpMeta `json:",inline"`
	ID     `json:"id"`
	Gid    uint32 `json:"gid"`
}

func (op *UpdateGidOp) Apply(s *DBState) error {
	n, ok := s.nodes[op.ID]
	if !ok {
		return util.ENOENT
	}
	switch n := n.(type) {
	case *FileNode:
		n.Gid = op.Gid
	case *DirNode:
		n.Gid = op.Gid
	default:
		return fmt.Errorf("UpdateGidOp: Unsupported node type: %d", n.GetType())
	}
	return nil
}

type UpdatePermModeOp struct {
	OpMeta   `json:",inline"`
	ID       `json:"id"`
	PermMode uint16 `json:"perm_mode"`
}

func (op *UpdatePermModeOp) Apply(s *DBState) error {
	n, ok := s.nodes[op.ID]
	if !ok {
		return util.ENOENT
	}

	switch n := n.(type) {
	case *FileNode:
		n.PermMode = op.PermMode
	case *DirNode:
		n.PermMode = op.PermMode
	default:
		return fmt.Errorf("UpdatePermModeOp: Unsupported node type: %d", n.GetType())
	}
	return nil
}

type UpdateModifiedTOp struct {
	OpMeta    `json:",inline"`
	ID        `json:"id"`
	ModifiedT time.Time `json:"modified_t"`
}

func (op *UpdateModifiedTOp) Apply(s *DBState) error {
	n, ok := s.nodes[op.ID]
	if !ok {
		return util.ENOENT
	}
	switch n := n.(type) {
	case *FileNode:
		n.ModifiedT = op.ModifiedT
	case *DirNode:
		n.ModifiedT = op.ModifiedT
	default:
		return fmt.Errorf("UpdateModifiedTOp: Unsupported node type: %d", n.GetType())
	}
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
		return util.ENOENT
	}
	dstn, ok := s.nodes[op.DstDirID]
	if !ok {
		return util.ENOENT
	}

	srcdn, ok := srcn.(*DirNode)
	if !ok {
		return util.ENOTDIR
	}
	dstdn, ok := dstn.(*DirNode)
	if !ok {
		return util.ENOTDIR
	}

	id, ok := srcdn.Entries[op.SrcName]
	if !ok {
		return util.ENOENT
	}

	if srcdn == dstdn && op.SrcName == op.DstName {
		return nil
	}

	mn, ok := s.nodes[id]
	if !ok {
		return fmt.Errorf("Rename target node id %d do not exist! Filesystem INodeDB corrupted?", id)
	}
	if mdn, ok := mn.(*DirNode); ok {
		mdn.ParentID = op.DstDirID
	}

	delete(srcdn.Entries, op.SrcName)
	now := time.Now()
	srcdn.ModifiedT = now
	dstdn.Entries[op.DstName] = id
	dstdn.ModifiedT = now
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
		return util.ENOENT
	}
	dn, ok := n.(*DirNode)
	if !ok {
		return util.ENOTDIR
	}

	tgtid, ok := dn.Entries[op.Name]
	if !ok {
		return util.ENOENT
	}
	if tgtnode, ok := s.nodes[tgtid]; ok {
		if tgtdirnode, ok := tgtnode.(*DirNode); ok {
			if len(tgtdirnode.Entries) != 0 {
				return util.ENOTEMPTY
			}
		}
	}

	delete(dn.Entries, op.Name)
	dn.ModifiedT = time.Now()
	return nil
}

type AlwaysFailForTestingOp struct {
	OpMeta `json:",inline"`
}

func (op *AlwaysFailForTestingOp) Apply(s *DBState) error {
	return errors.New("Forced fail for testing")
}
