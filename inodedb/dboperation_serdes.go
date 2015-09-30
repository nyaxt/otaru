package inodedb

import (
	"fmt"

	"encoding/json"
)

func SetOpMeta(op DBOperation) error {
	switch op.(type) {
	case *InitializeFileSystemOp:
		op.(*InitializeFileSystemOp).Kind = "InitializeFileSystemOp"
	case *CreateNodeOp:
		op.(*CreateNodeOp).Kind = "CreateNodeOp"
	case *HardLinkOp:
		op.(*HardLinkOp).Kind = "HardLinkOp"
	case *UpdateChunksOp:
		op.(*UpdateChunksOp).Kind = "UpdateChunksOp"
	case *UpdateSizeOp:
		op.(*UpdateSizeOp).Kind = "UpdateSizeOp"
	case *UpdateUidOp:
		op.(*UpdateUidOp).Kind = "UpdateUidOp"
	case *UpdateGidOp:
		op.(*UpdateGidOp).Kind = "UpdateGidOp"
	case *UpdatePermModeOp:
		op.(*UpdatePermModeOp).Kind = "UpdatePermModeOp"
	case *UpdateModifiedTOp:
		op.(*UpdateModifiedTOp).Kind = "UpdateModifiedTOp"
	case *RenameOp:
		op.(*RenameOp).Kind = "RenameOp"
	case *RemoveOp:
		op.(*RemoveOp).Kind = "RemoveOp"
	case *AlwaysFailForTestingOp:
		op.(*AlwaysFailForTestingOp).Kind = "AlwaysFailForTestingOp"
	default:
		return fmt.Errorf("Encoder undefined for op: %v", op)
	}
	return nil
}

func SetOpMetas(ops []DBOperation) error {
	for _, op := range ops {
		if err := SetOpMeta(op); err != nil {
			return err
		}
	}
	return nil
}

func EncodeDBOperationsToJson(ops []DBOperation) ([]byte, error) {
	if err := SetOpMetas(ops); err != nil {
		return nil, err
	}
	return json.Marshal(ops)
}

type UnresolvedDBTransaction struct {
	TxID `json:"txid"`
	Ops  []*json.RawMessage
}

func DecodeDBOperationsFromJson(jsonb []byte) ([]DBOperation, error) {
	var msgs []*json.RawMessage
	if err := json.Unmarshal(jsonb, &msgs); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal messages: %v", err)
	}

	return ResolveDBOperations(msgs)
}

func ResolveDBTransaction(utx UnresolvedDBTransaction) (DBTransaction, error) {
	ops, err := ResolveDBOperations(utx.Ops)
	if err != nil {
		return DBTransaction{}, nil
	}

	return DBTransaction{TxID: utx.TxID, Ops: ops}, nil
}

func ResolveDBOperations(msgs []*json.RawMessage) ([]DBOperation, error) {
	ops := make([]DBOperation, 0, len(msgs))
	for _, msg := range msgs {
		var meta OpMeta
		if err := json.Unmarshal([]byte(*msg), &meta); err != nil {
			return nil, fmt.Errorf("Failed to unmarshal OpMeta: %v", err)
		}

		switch meta.Kind {
		case "InitializeFileSystemOp":
			var op InitializeFileSystemOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		case "CreateNodeOp":
			var op CreateNodeOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		case "HardLinkOp":
			var op HardLinkOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		case "UpdateChunksOp":
			var op UpdateChunksOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		case "UpdateSizeOp":
			var op UpdateSizeOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		case "UpdateUidOp":
			var op UpdateUidOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		case "UpdateGidOp":
			var op UpdateGidOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		case "UpdatePermModeOp":
			var op UpdatePermModeOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		case "UpdateModifiedTOp":
			var op UpdateModifiedTOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		case "RenameOp":
			var op RenameOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		case "RemoveOp":
			var op RemoveOp
			if err := json.Unmarshal([]byte(*msg), &op); err != nil {
				return nil, err
			}
			ops = append(ops, &op)
		default:
			return nil, fmt.Errorf("Unknown kind \"%s\"", meta.Kind)
		}
	}
	return ops, nil
}
