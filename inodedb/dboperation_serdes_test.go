package inodedb_test

import (
	"testing"

	i "github.com/nyaxt/otaru/inodedb"
)

func TestEncodeDBOperationToJson_InitializeFileSystemOp(t *testing.T) {
	json, err := i.EncodeDBOperationsToJson([]i.DBOperation{&i.InitializeFileSystemOp{}})
	if err != nil {
		t.Errorf("EncodeDBOperationToJson failed: %v", err)
		return
	}

	// t.Errorf("%v", string(json))
	ops, err := i.DecodeDBOperationsFromJson(json)
	if err != nil {
		t.Errorf("DecodeDBOperationToJson failed: %v", err)
		return
	}
	if _, ok := ops[0].(*i.InitializeFileSystemOp); !ok {
		t.Errorf("Decode failed to recover original type")
	}
}

func TestEncodeDBOperationToJson_CreateNodeOp(t *testing.T) {
	json, err := i.EncodeDBOperationsToJson([]i.DBOperation{&i.CreateNodeOp{
		NodeLock: i.NodeLock{ID: 123, Ticket: 456},
		OrigPath: "/foo/bar",
		Type:     i.DirNodeT,
	}})
	if err != nil {
		t.Errorf("EncodeDBOperationToJson failed: %v", err)
		return
	}

	ops, err := i.DecodeDBOperationsFromJson(json)
	if err != nil {
		t.Errorf("DecodeDBOperationToJson failed: %v", err)
		return
	}

	dirop, ok := ops[0].(*i.CreateNodeOp)
	if !ok {
		t.Errorf("Decode failed to recover original type")
	}

	if dirop.NodeLock.ID != 123 {
		t.Errorf("encode/decode data mismatch")
	}
	if dirop.NodeLock.Ticket != 456 {
		t.Errorf("encode/decode data mismatch")
	}
	if dirop.OrigPath != "/foo/bar" {
		t.Errorf("encode/decode data mismatch")
	}
}
