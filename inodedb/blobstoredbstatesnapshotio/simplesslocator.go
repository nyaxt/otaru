package blobstoredbstatesnapshotio

import (
	"fmt"
	"github.com/nyaxt/otaru/metadata"
)

func generateBlobpath() string {
	return fmt.Sprintf("%s_SimpleSSLocator", metadata.INodeDBSnapshotBlobpathPrefix)
}

type SimpleSSLocator struct{}

func (SimpleSSLocator) Locate(history int) (string, error) {
	return generateBlobpath(), nil
}

func (SimpleSSLocator) GenerateBlobpath() string {
	return generateBlobpath()
}

func (SimpleSSLocator) Put(blobpath string, txid int64) error {
	return nil
}
