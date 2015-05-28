package inodedb

import (
	"fmt"

	"encoding/json"
)

func EncodeDBOperationToJson(op DBOperation) ([]byte, error) {
	switch op.(type) {
	case *InitializeFileSystemOp:
		op.(*InitializeFileSystemOp).Kind = "InitializeFileSystemOp"
	case *CreateDirOp:
		op.(*CreateDirOp).Kind = "CreateDirOp"
	case *CreateFileOp:
		op.(*CreateFileOp).Kind = "CreateFileOp"
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
	default:
		return nil, fmt.Errorf("Encoder undefined for op: %v", op)
	}

	return json.Marshal(op)
}

func DecodeDBOperationFromJson(jsonb []byte) (DBOperation, error) {
	var meta OpMeta
	if err := json.Unmarshal(jsonb, &meta); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal OpMeta: %v", err)
	}

	switch meta.Kind {
	case "InitializeFileSystemOp":
		var op InitializeFileSystemOp
		if err := json.Unmarshal(jsonb, &op); err != nil {
			return nil, err
		}
		return &op, nil
	case "CreateDirOp":
		var op CreateDirOp
		if err := json.Unmarshal(jsonb, &op); err != nil {
			return nil, err
		}
		return &op, nil
	case "CreateFileOp":
		var op CreateFileOp
		if err := json.Unmarshal(jsonb, &op); err != nil {
			return nil, err
		}
		return &op, nil
	case "HardLinkOp":
		var op HardLinkOp
		if err := json.Unmarshal(jsonb, &op); err != nil {
			return nil, err
		}
		return &op, nil
	case "UpdateChunksOp":
		var op UpdateChunksOp
		if err := json.Unmarshal(jsonb, &op); err != nil {
			return nil, err
		}
		return &op, nil
	case "UpdateSizeOp":
		var op UpdateSizeOp
		if err := json.Unmarshal(jsonb, &op); err != nil {
			return nil, err
		}
		return &op, nil
	case "RenameOp":
		var op RenameOp
		if err := json.Unmarshal(jsonb, &op); err != nil {
			return nil, err
		}
		return &op, nil
	case "RemoveOp":
		var op RemoveOp
		if err := json.Unmarshal(jsonb, &op); err != nil {
			return nil, err
		}
		return &op, nil
	default:
		return nil, fmt.Errorf("Unknown kind \"%s\"", meta.Kind)
	}
}
