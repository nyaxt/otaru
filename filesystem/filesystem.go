package filesystem

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/nyaxt/otaru/blobstore"
	"github.com/nyaxt/otaru/btncrypt"
	"github.com/nyaxt/otaru/chunkstore"
	"github.com/nyaxt/otaru/filewritecache"
	fl "github.com/nyaxt/otaru/flags"
	"github.com/nyaxt/otaru/inodedb"
	"github.com/nyaxt/otaru/util"
)

type FileSystem struct {
	idb inodedb.DBHandler

	bs blobstore.RandomAccessBlobStore
	c  *btncrypt.Cipher

	muOpenFiles sync.Mutex
	openFiles   map[inodedb.ID]*OpenFile

	muOrigPath sync.Mutex
	origpath   map[inodedb.ID]string

	logger *zap.Logger
}

func NewFileSystem(idb inodedb.DBHandler, bs blobstore.RandomAccessBlobStore, c *btncrypt.Cipher, logger *zap.Logger) *FileSystem {
	fs := &FileSystem{
		idb: idb,
		bs:  bs,
		c:   c,

		openFiles: make(map[inodedb.ID]*OpenFile),
		origpath:  make(map[inodedb.ID]string),

		logger: logger.Named("filesystem"),
	}
	fs.setOrigPathForId(inodedb.RootDirID, "/")

	return fs
}

type FileSystemStats struct {
	NumOpenFiles int `json:"num_open_files"`
	NumOrigPath  int `json:"num_orig_path"`
}

func (fs *FileSystem) GetStats() (ret FileSystemStats) {
	fs.muOpenFiles.Lock()
	ret.NumOpenFiles = len(fs.openFiles)
	fs.muOpenFiles.Unlock()

	fs.muOrigPath.Lock()
	ret.NumOrigPath = len(fs.origpath)
	fs.muOrigPath.Unlock()

	return
}

func (fs *FileSystem) tryGetOrigPath(id inodedb.ID) string {
	fs.muOrigPath.Lock()
	defer fs.muOrigPath.Unlock()

	origpath, ok := fs.origpath[id]
	if !ok {
		fs.logger.Sugar().Warnf("Failed to lookup orig path for ID %d", id)
		return "<unknown>"
	}
	// fs.logger.Sugar().Warnf("Orig path for ID %d is \"%s\"", id, origpath)
	return origpath
}

func (fs *FileSystem) setOrigPathForId(id inodedb.ID, origpath string) {
	fs.muOrigPath.Lock()
	defer fs.muOrigPath.Unlock()

	if len(origpath) == 0 {
		delete(fs.origpath, id)
	}
	fs.origpath[id] = origpath
}

func (fs *FileSystem) snapshotOpenFiles() []*OpenFile {
	fs.muOpenFiles.Lock()
	defer fs.muOpenFiles.Unlock()

	ret := make([]*OpenFile, 0, len(fs.openFiles))
	for _, of := range fs.openFiles {
		ret = append(ret, of)
	}

	return ret
}

func (fs *FileSystem) Sync() error {
	var me error
	if s, ok := fs.idb.(util.Syncer); ok {
		if err := s.Sync(); err != nil {
			me = multierr.Append(me, fmt.Errorf("Failed to sync INodeDB: %v", err))
		}
	}

	ofss := fs.snapshotOpenFiles()
	for _, of := range ofss {
		of.Sync()
	}

	return me
}

func (fs *FileSystem) TotalSize() (int64, error) {
	tsizer, ok := fs.bs.(blobstore.TotalSizer)
	if !ok {
		return 0, fmt.Errorf("Backend blobstore doesn't support TotalSize()")
	}
	return tsizer.TotalSize()
}

func (fs *FileSystem) ParentID(id inodedb.ID) (inodedb.ID, error) {
	v, _, err := fs.idb.QueryNode(id, false)
	if err != nil {
		return inodedb.RootDirID, err
	}
	if v.GetType() != inodedb.DirNodeT {
		return inodedb.RootDirID, util.ENOTDIR
	}

	dv := v.(*inodedb.DirNodeView)
	return dv.ParentID, err
}

func (fs *FileSystem) DirEntries(id inodedb.ID) (map[string]inodedb.ID, error) {
	v, _, err := fs.idb.QueryNode(id, false)
	if err != nil {
		return nil, err
	}
	if v.GetType() != inodedb.DirNodeT {
		return nil, util.ENOTDIR
	}

	dv := v.(*inodedb.DirNodeView)

	dirorigpath := fs.tryGetOrigPath(id)
	for name, id := range dv.Entries {
		fs.setOrigPathForId(id, fmt.Sprintf("%s/%s", dirorigpath, name))
	}

	return dv.Entries, err
}

func (fs *FileSystem) Rename(srcDirID inodedb.ID, srcName string, dstDirID inodedb.ID, dstName string) error {
	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.RenameOp{
			SrcDirID: srcDirID, SrcName: srcName,
			DstDirID: dstDirID, DstName: dstName,
		},
	}}
	if _, err := fs.idb.ApplyTransaction(tx); err != nil {
		return err
	}

	// FIXME: fs.setOrigPathForId

	return nil
}

func (fs *FileSystem) Remove(dirID inodedb.ID, name string) error {
	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.RemoveOp{
			NodeLock: inodedb.NodeLock{dirID, inodedb.NoTicket}, Name: name,
		},
	}}
	if _, err := fs.idb.ApplyTransaction(tx); err != nil {
		return err
	}

	// FIXME: fs.setOrigPathForId

	return nil
}

func (fs *FileSystem) createNode(dirID inodedb.ID, name string, typ inodedb.Type, permmode uint16, uid, gid uint32, modifiedT time.Time) (inodedb.ID, error) {
	nlock, err := fs.idb.LockNode(inodedb.AllocateNewNodeID)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := fs.idb.UnlockNode(nlock); err != nil {
			fs.logger.Sugar().Warnf("Failed to unlock node when creating file: %v", err)
		}
	}()

	dirorigpath := fs.tryGetOrigPath(dirID)
	origpath := fmt.Sprintf("%s/%s", dirorigpath, name)

	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.CreateNodeOp{NodeLock: nlock, OrigPath: origpath, ParentID: dirID, Type: typ, PermMode: permmode, Uid: uid, Gid: gid, ModifiedT: modifiedT},
		&inodedb.HardLinkOp{NodeLock: inodedb.NodeLock{dirID, inodedb.NoTicket}, Name: name, TargetID: nlock.ID},
	}}
	if _, err := fs.idb.ApplyTransaction(tx); err != nil {
		return 0, err
	}

	fs.setOrigPathForId(nlock.ID, origpath)

	return nlock.ID, nil
}

func (fs *FileSystem) CreateFile(dirID inodedb.ID, name string, permmode uint16, uid, gid uint32, modifiedT time.Time) (inodedb.ID, error) {
	return fs.createNode(dirID, name, inodedb.FileNodeT, permmode, uid, gid, modifiedT)
}

func (fs *FileSystem) CreateDir(dirID inodedb.ID, name string, permmode uint16, uid, gid uint32, modifiedT time.Time) (inodedb.ID, error) {
	return fs.createNode(dirID, name, inodedb.DirNodeT, permmode, uid, gid, modifiedT)
}

type Attr struct {
	ID   inodedb.ID   `json:"id"`
	Type inodedb.Type `json:"type"`
	Size int64        `json:"size"`

	OrigPath  string    `json:"orig_path"`
	Uid       uint32    `json:"uid"`
	Gid       uint32    `json:"gid"`
	PermMode  uint16    `json:"mode_perm"`
	ModifiedT time.Time `json:"modified_t"`
}

func (fs *FileSystem) Attr(id inodedb.ID) (Attr, error) {
	v, _, err := fs.idb.QueryNode(id, false)
	if err != nil {
		return Attr{}, err
	}

	size := int64(0)
	if fn, ok := v.(*inodedb.FileNodeView); ok {
		size = fn.Size
	}

	a := Attr{
		ID:   v.GetID(),
		Type: v.GetType(),
		Size: size,

		OrigPath:  v.GetOrigPath(),
		Uid:       v.GetUid(),
		Gid:       v.GetGid(),
		PermMode:  fl.MaskPermMode(v.GetPermMode(), fs.bs.Flags()),
		ModifiedT: v.GetModifiedT(),
	}
	return a, nil
}

type ValidAttrFields uint32

const (
	UidValid ValidAttrFields = 1 << iota
	GidValid
	PermModeValid
	ModifiedTValid
)

func (valid ValidAttrFields) String() string {
	var b bytes.Buffer

	if valid&UidValid != 0 {
		b.WriteString("UidValid|")
	}
	if valid&GidValid != 0 {
		b.WriteString("GidValid|")
	}
	if valid&PermModeValid != 0 {
		b.WriteString("PermModeValid|")
	}
	if valid&ModifiedTValid != 0 {
		b.WriteString("ModifiedTValid|")
	}
	// trim last "|"
	if b.Len() > 0 {
		b.Truncate(b.Len() - 1)
	}

	return b.String()
}

func (fs *FileSystem) SetAttr(id inodedb.ID, a Attr, valid ValidAttrFields) error {
	fs.logger.Sugar().Infof("SetAttr id: %d, a: %+v, valid: %s", id, a, valid)

	ops := make([]inodedb.DBOperation, 0, 4)
	if valid&UidValid != 0 {
		ops = append(ops, &inodedb.UpdateUidOp{ID: id, Uid: a.Uid})
	}
	if valid&GidValid != 0 {
		ops = append(ops, &inodedb.UpdateGidOp{ID: id, Gid: a.Gid})
	}
	if valid&PermModeValid != 0 {
		ops = append(ops, &inodedb.UpdatePermModeOp{ID: id, PermMode: a.PermMode})
	}
	if valid&ModifiedTValid != 0 {
		ops = append(ops, &inodedb.UpdateModifiedTOp{ID: id, ModifiedT: a.ModifiedT})
	}

	if _, err := fs.idb.ApplyTransaction(inodedb.DBTransaction{Ops: ops}); err != nil {
		return err
	}
	return nil
}

func (fs *FileSystem) IsDir(id inodedb.ID) (bool, error) {
	v, _, err := fs.idb.QueryNode(id, false)
	if err != nil {
		return false, err
	}

	return v.GetType() == inodedb.DirNodeT, nil
}

type FileHandle struct {
	of    *OpenFile
	flags int
}

type OpenFile struct {
	fs    *FileSystem
	nlock inodedb.NodeLock
	wc    *filewritecache.FileWriteCache
	cfio  *chunkstore.ChunkedFileIO

	origFilename string

	handles []*FileHandle

	mu sync.Mutex

	logger *zap.Logger
}

func (fs *FileSystem) getOrCreateOpenFile(id inodedb.ID) *OpenFile {
	fs.muOpenFiles.Lock()
	defer fs.muOpenFiles.Unlock()

	of, ok := fs.openFiles[id]
	if ok {
		return of
	}
	of = &OpenFile{
		fs: fs,
		wc: filewritecache.New(),

		handles: make([]*FileHandle, 0, 1),

		logger: fs.logger.Named("OpenFile").With(zap.Uint64("id", uint64(id))),
	}
	fs.openFiles[id] = of
	return of
}

func (fs *FileSystem) OpenFile(id inodedb.ID, flags int) (*FileHandle, error) {
	s := fs.logger.Sugar()
	s.Infof("OpenFile(id: %v, flags rok: %t wok: %t)", id, fl.IsReadAllowed(flags), fl.IsWriteAllowed(flags))

	tryLock := fl.IsWriteAllowed(flags)
	if tryLock && !fl.IsWriteAllowed(fs.bs.Flags()) {
		return nil, util.EACCES
	}

	of := fs.getOrCreateOpenFile(id)

	of.mu.Lock()
	defer of.mu.Unlock()

	ofIsInitialized := of.nlock.ID != 0
	if ofIsInitialized && (of.nlock.HasTicket() || !tryLock) {
		// No need to upgrade lock. Just use cached filehandle.
		s.Infof("Using cached of for inode id: %v", id)
		return of.OpenHandleWithoutLock(flags), nil
	}

	// upgrade lock or acquire new lock...
	v, nlock, err := fs.idb.QueryNode(id, tryLock)
	if err != nil {
		return nil, err
	}
	if v.GetType() != inodedb.FileNodeT {
		if err := fs.idb.UnlockNode(nlock); err != nil {
			s.Warnf("Unlock node failed for non-file node: %v", err)
		}

		if v.GetType() == inodedb.DirNodeT {
			return nil, util.EISDIR
		}
		return nil, fmt.Errorf("Specified node not file but has type %v", v.GetType())
	}

	of.nlock = nlock
	caio := NewINodeDBChunksArrayIO(fs.idb, nlock)
	of.cfio = chunkstore.NewChunkedFileIO(fs.bs, fs.c, caio)
	of.cfio.SetOrigFilename(fs.tryGetOrigPath(nlock.ID))

	if fl.IsWriteTruncate(flags) {
		if err := of.truncateWithLock(0); err != nil {
			return nil, fmt.Errorf("Failed to truncate file: %v", err)
		}
	}

	fh := of.OpenHandleWithoutLock(flags)
	return fh, nil
}

func (fs *FileSystem) closeOpenFile(id inodedb.ID) {
	fs.muOpenFiles.Lock()
	defer fs.muOpenFiles.Unlock()

	delete(fs.openFiles, id)
}

func (fs *FileSystem) TruncateFile(id inodedb.ID, newsize int64) error {
	fh, err := fs.OpenFile(id, fl.O_WRONLY)
	if err != nil {
		return fmt.Errorf("Failed to OpenFile: %v", err)
	}
	defer fh.Close()

	return fh.Truncate(newsize)
}

func (fs *FileSystem) SyncFile(id inodedb.ID) error {
	if !fl.IsWriteAllowed(fs.bs.Flags()) {
		// no need to sync if fs is readonly
		return nil
	}

	fh, err := fs.OpenFile(id, fl.O_WRONLY)
	if err != nil {
		return fmt.Errorf("Failed to OpenFile: %v", err)
	}
	defer fh.Close()

	return fh.Sync()
}

func (of *OpenFile) OpenHandleWithoutLock(flags int) *FileHandle {
	fh := &FileHandle{of: of, flags: flags}
	of.handles = append(of.handles, fh)
	return fh
}

func (of *OpenFile) CloseHandle(tgt *FileHandle) {
	s := of.logger.Sugar()
	if tgt.of == nil {
		s.Warnf("Detected FileHandle double close!")
		return
	}
	if tgt.of != of {
		s.Errorf("Attempt to close handle for other OpenFile. tgt fh: %+v, of: %+v", tgt, of)
		return
	}

	wasWriteHandle := fl.IsWriteAllowed(tgt.flags)
	ofHasOtherWriteHandle := false

	tgt.of = nil

	of.mu.Lock()
	ul := util.EnsureUnlocker{&of.mu}
	defer ul.Unlock()

	// remove tgt from of.handles slice
	newHandles := make([]*FileHandle, 0, len(of.handles)-1)
	for _, h := range of.handles {
		if h != tgt {
			if fl.IsWriteAllowed(h.flags) {
				ofHasOtherWriteHandle = true
			}
			newHandles = append(newHandles, h)
		}
	}
	of.handles = newHandles

	wasLastWriteHandle := wasWriteHandle && !ofHasOtherWriteHandle
	if wasLastWriteHandle {
		if err := of.wc.Sync(of.cfio); err != nil {
			s.Errorf("FileWriteCache sync failed: %v", err)
		}

		// Note: if len(of.handles) == 0, below will create cfio just to be closed immediately below
		// This should be ok, as instantiate -> Close() unused cfio is lightweight.
		of.downgradeToReadLock()
	}

	if len(of.handles) == 0 {
		if err := of.cfio.Close(); err != nil {
			s.Warnf("Closing ChunkedFileIO when all handles closed failed: %v", err)
		}

		id := of.nlock.ID
		ul.Unlock()

		of.fs.closeOpenFile(id)
	}
}

func (of *OpenFile) downgradeToReadLock() {
	s := of.logger.Sugar()

	s.Infof("Downgrade %v to read lock.", of)
	// Note: assumes of.mu is Lock()-ed

	if !of.nlock.HasTicket() {
		s.Warnf("Attempt to downgrade node lock, but no excl lock found. of: %v", of)
		return
	}

	if err := of.fs.idb.UnlockNode(of.nlock); err != nil {
		s.Warnf("Unlocking node to downgrade to read lock failed: %v", err)
	}
	of.nlock.Ticket = inodedb.NoTicket

	if err := of.cfio.Close(); err != nil {
		s.Warnf("Closing ChunkedFileIO when downgrading to read lock failed: %v", err)
	}

	caio := NewINodeDBChunksArrayIO(of.fs.idb, of.nlock)
	of.cfio = chunkstore.NewChunkedFileIO(of.fs.bs, of.fs.c, caio)
}

func (of *OpenFile) updateModifiedTWithoutLock() error {
	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.UpdateModifiedTOp{ID: of.nlock.ID, ModifiedT: time.Now()},
	}}
	if _, err := of.fs.idb.ApplyTransaction(tx); err != nil {
		return fmt.Errorf("Failed to update ModifiedT size: %v", err)
	}
	return nil
}

func (of *OpenFile) updateSizeWithoutLock(newsize int64) error {
	tx := inodedb.DBTransaction{Ops: []inodedb.DBOperation{
		&inodedb.UpdateSizeOp{NodeLock: of.nlock, Size: newsize},
	}}
	if _, err := of.fs.idb.ApplyTransaction(tx); err != nil {
		return fmt.Errorf("Failed to update FileNode size: %v", err)
	}
	return nil
}

// sizeMayFailWithoutLock returns file size if succeed. The size query may fail with an error.
func (of *OpenFile) sizeMayFailWithoutLock() (int64, error) {
	v, _, err := of.fs.idb.QueryNode(of.nlock.ID, false)
	if err != nil {
		return 0, fmt.Errorf("Failed to QueryNode inodedb: %v", err)
	}
	fv, ok := v.(*inodedb.FileNodeView)
	if !ok {
		return 0, fmt.Errorf("Non-FileNodeView returned from QueryNode. Type: %v", v.GetType())
	}
	return fv.Size, nil
}

func (of *OpenFile) PWrite(p []byte, offset int64) error {
	of.mu.Lock()
	defer of.mu.Unlock()

	currentSize, err := of.sizeMayFailWithoutLock()
	if err != nil {
		return err
	}

	if err := of.wc.PWrite(p, offset); err != nil {
		return err
	}

	if of.wc.NeedsSync() {
		if err := of.wc.Sync(of.cfio); err != nil {
			return err
		}
	}

	right := offset + int64(len(p))
	if right > currentSize {
		if err := of.updateSizeWithoutLock(right); err != nil {
			return err
		}
	} else {
		if err := of.updateModifiedTWithoutLock(); err != nil {
			return err
		}
	}

	return nil
}

func (of *OpenFile) Append(p []byte) error {
	of.mu.Lock()
	defer of.mu.Unlock()

	currentSize, err := of.sizeMayFailWithoutLock()
	if err != nil {
		return err
	}

	if err := of.wc.PWrite(p, currentSize); err != nil {
		return err
	}

	if of.wc.NeedsSync() {
		if err := of.wc.Sync(of.cfio); err != nil {
			return err
		}
	}

	right := currentSize + int64(len(p))
	if err := of.updateSizeWithoutLock(right); err != nil {
		return err
	}

	return nil
}

func (of *OpenFile) ReadAt(p []byte, offset int64) (int, error) {
	of.mu.Lock()
	defer of.mu.Unlock()

	return of.wc.ReadAtThrough(p, offset, of.cfio)
}

func (of *OpenFile) Sync() error {
	of.mu.Lock()
	defer of.mu.Unlock()

	if err := of.wc.Sync(of.cfio); err != nil {
		return fmt.Errorf("FileWriteCache sync failed: %v", err)
	}
	return nil
}

func (of *OpenFile) Size() int64 {
	of.mu.Lock()
	defer of.mu.Unlock()

	size, err := of.sizeMayFailWithoutLock()
	if err != nil {
		of.logger.Sugar().Warnf("Failed to query OpenFile.Size(), but suppressing error: %v", err)
		return 0
	}
	return size
}

func (of *OpenFile) Truncate(newsize int64) error {
	of.mu.Lock()
	defer of.mu.Unlock()

	return of.truncateWithLock(newsize)
}

func (of *OpenFile) truncateWithLock(newsize int64) error {
	oldsize, err := of.sizeMayFailWithoutLock()
	if err != nil {
		return err
	}

	if newsize > oldsize {
		return of.updateSizeWithoutLock(newsize)
	} else if newsize < oldsize {
		of.wc.Truncate(newsize)
		of.cfio.Truncate(newsize)
		return of.updateSizeWithoutLock(newsize)
	} else {
		return nil
	}
}

func (fh *FileHandle) ID() inodedb.ID {
	return fh.of.nlock.ID
}

func (fh *FileHandle) PWrite(p []byte, offset int64) error {
	if !fl.IsWriteAllowed(fh.flags) {
		return util.EBADF
	}

	if fl.IsWriteAppend(fh.flags) {
		return fh.of.Append(p)
	}

	return fh.of.PWrite(p, offset)
}

func (fh *FileHandle) ReadAt(p []byte, offset int64) (int, error) {
	if !fl.IsReadAllowed(fh.flags) {
		return 0, util.EBADF
	}

	return fh.of.ReadAt(p, offset)
}

func (fh *FileHandle) Sync() error {
	if !fl.IsWriteAllowed(fh.flags) {
		return nil
	}

	return fh.of.Sync()
}

func (fh *FileHandle) Size() int64 {
	return fh.of.Size()
}

func (fh *FileHandle) Truncate(newsize int64) error {
	if !fl.IsWriteAllowed(fh.flags) {
		return util.EBADF
	}

	return fh.of.Truncate(newsize)
}

func (fh *FileHandle) Close() {
	fh.of.CloseHandle(fh)
}

func (fh *FileHandle) Attr() (Attr, error) {
	return fh.of.fs.Attr(fh.ID())
}
