package otaru_test

import (
	. "github.com/nyaxt/otaru"

	"bytes"
	"reflect"
	"testing"
)

func TestINodeDB_SerializeSnapshot(t *testing.T) {
	idb := NewINodeDB()
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
