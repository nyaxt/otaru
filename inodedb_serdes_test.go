package otaru_test

import (
	. "github.com/nyaxt/otaru"
	. "github.com/nyaxt/otaru/testutils"

	"bytes"
	"reflect"
	"testing"
)

func TestINodeDB_SerializeSnapshot(t *testing.T) {
	idb := NewINodeDBEmpty()
	NewDirNode(idb, "/")
	NewFileNode(idb, "/hoge.txt")
	NewFileNode(idb, "/fuga.txt")

	var buf bytes.Buffer
	if err := idb.SerializeSnapshot(&buf); err != nil {
		t.Errorf("SerializeSnapshot failed: %v", err)
	}

	idb2, err := DeserializeINodeDBSnapshot(&buf)
	if err != nil {
		t.Errorf("DeserializeINodeDBSnapshot failed: %v", err)
	}

	if !reflect.DeepEqual(idb, idb2) {
		t.Errorf("serdes mismatch!")
	}
}

func TestINodeDB_SaveLoadBlobStore_Empty(t *testing.T) {
	bs := TestFileBlobStore()
	{
		idb := NewINodeDBEmpty()
		if err := idb.SaveToBlobStore(bs, TestCipher()); err != nil {
			t.Errorf("Failed save: %v", err)
		}
	}
	{
		idb, err := LoadINodeDBFromBlobStore(bs, TestCipher())
		if err != nil {
			t.Errorf("Failed load: %v", err)
		}
		NewFileNode(idb, "/piyo.txt")
	}
}
