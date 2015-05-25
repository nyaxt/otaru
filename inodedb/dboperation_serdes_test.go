package inodedb_test

import (
	"testing"

	i "github.com/nyaxt/otaru/inodedb"
)

func TestEncodeDBOperationToJson_InitializeFileSystemOp(t *testing.T) {
	json, err := i.EncodeDBOperationToJson(&i.InitializeFileSystemOp{})
	if err != nil {
		t.Errorf("EncodeDBOperationToJson failed: %v", err)
		return
	}

	// t.Errorf("%v", string(json))
	op, err := i.DecodeDBOperationFromJson(json)
	if err != nil {
		t.Errorf("DecodeDBOperationToJson failed: %v", err)
		return
	}
	if _, ok := op.(*i.InitializeFileSystemOp); !ok {
		t.Errorf("Decode failed to recover original type")
	}
}

func TestEncodeDBOperationToJson_CreateDirOp(t *testing.T) {
	json, err := i.EncodeDBOperationToJson(&i.CreateDirOp{
		NodeLock: i.NodeLock{ID: 123, Ticket: 456},
		OrigPath: "/foo/bar",
	})
	if err != nil {
		t.Errorf("EncodeDBOperationToJson failed: %v", err)
		return
	}

	op, err := i.DecodeDBOperationFromJson(json)
	if err != nil {
		t.Errorf("DecodeDBOperationToJson failed: %v", err)
		return
	}

	dirop, ok := op.(*i.CreateDirOp)
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
