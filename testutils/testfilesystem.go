package testutils

import (
	"log"

	"go.uber.org/zap"

	"github.com/nyaxt/otaru/filesystem"
	"github.com/nyaxt/otaru/inodedb"
)

func TestINodeDB() inodedb.DBHandler {
	snapshotio := inodedb.NewSimpleDBStateSnapshotIO()
	txio := inodedb.NewSimpleDBTransactionLogIO()
	idb, err := inodedb.NewEmptyDB(snapshotio, txio)
	if err != nil {
		log.Panicf("NewEmptyDB failed: %v", err)
	}

	return idb
}

func TestFileSystem() *filesystem.FileSystem {
	EnsureLogger()

	idb := TestINodeDB()
	bs := TestFileBlobStore()
	return filesystem.NewFileSystem(idb, bs, TestCipher(), zap.L())
}
