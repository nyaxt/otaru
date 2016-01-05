package blobstoredbstatesnapshotio

import (
	"fmt"
	"github.com/nyaxt/otaru/metadata"
)

func generateBlobpath() string {
	return fmt.Sprintf("%s_SimpleSSLocator", metadata.INodeDBSnapshotBlobpathPrefix)
}

var simplesslocatorTxID int64

type SimpleSSLocator struct{}

func (SimpleSSLocator) Locate(history int) (string, int64, error) {
	return generateBlobpath(), simplesslocatorTxID, nil
}

func (SimpleSSLocator) GenerateBlobpath() string {
	return generateBlobpath()
}

func (SimpleSSLocator) Put(blobpath string, txid int64) error {
	simplesslocatorTxID = txid
	return nil
}
