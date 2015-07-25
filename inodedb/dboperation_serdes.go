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

func DecodeDBOperationsFromJson(jsonb []byte) ([]DBOperation, error) {
	var msgs []*json.RawMessage
	if err := json.Unmarshal(jsonb, &msgs); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal messages: %v", err)
	}

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
