package webdav

import (
	"time"

	"github.com/nyaxt/otaru/pb"
)

type Entry struct {
	Id           uint64
	Name         string
	Size         int64
	ModifiedTime time.Time
	IsDir        bool
}

func INodeViewToEntry(v *pb.INodeView) *Entry {
	return &Entry{
		Id:           v.Id,
		Name:         v.Name,
		Size:         v.Size,
		ModifiedTime: time.Unix(v.ModifiedTime, 0),
		IsDir:        v.Type == pb.INodeType_DIR,
	}
}

type Listing []*Entry
