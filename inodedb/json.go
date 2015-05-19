package inodedb

import (
// "encoding/json"
)

type JSONEncodable interface {
	MarshalJSON() ([]byte, error)
}
