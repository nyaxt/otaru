package logbuf

import (
	"fmt"
	"os"

	"time"

	"github.com/nyaxt/otaru/logger"
	"github.com/nyaxt/otaru/util"
)

type Entry struct {
	Id       int
	Log      string
	Category string
	logger.Level
	time.Time
	Location string
}

type LogBuf struct {
	entries    []*Entry
	maxEntries int
	nextId     int
}

var _ = logger.Logger(&LogBuf{})

func NewLogBuf(maxEntries int) *LogBuf {
	if maxEntries < 1 {
		panic("NewLogBuf maxEntries must be larger than 0")
	}

	return &LogBuf{
		entries:    make([]*Entry, 0, maxEntries),
		maxEntries: maxEntries,
		nextId:     0,
	}
}

func (lb *LogBuf) Log(lv logger.Level, data map[string]interface{}) {
	if len(lb.entries) == lb.maxEntries-1 {
		lb.entries = lb.entries[1:]
	}

	category := ""
	if c, ok := data["category"]; ok {
		category = c.(string)
	}
	entry := &Entry{
		Id:       lb.nextId,
		Log:      data["log"].(string),
		Category: category,
		Level:    lv,
		Time:     data["time"].(time.Time),
		Location: data["location"].(string),
	}
	lb.nextId++

	lb.entries = append(lb.entries, entry)
}

func (lb *LogBuf) WillAccept(lv logger.Level) bool { return true }

func (lb *LogBuf) Query(minId int, categories []string, limit int) []*Entry {
	if len(lb.entries) == 0 {
		return []*Entry{}
	}
	lbMinId := lb.entries[0].Id
	i := minId - lbMinId
	n := util.IntMin(limit, len(lb.entries)-i)
	if i <= 0 {
		n += i
		i = 0
	}
	if n <= 0 {
		return []*Entry{}
	}

	if len(categories) == 0 {
		fmt.Fprintf(os.Stderr, "len %d minid %d i %d n %d\n", len(lb.entries), lbMinId, i, n)
		return lb.entries[i : i+n]
	}

	cmap := make(map[string]struct{}, len(categories))
	for _, c := range categories {
		cmap[c] = struct{}{}
	}

	ret := make([]*Entry, 0, n)
	for ; i < len(lb.entries); i++ {
		e := lb.entries[i]
		if _, ok := cmap[e.Category]; ok {
			ret = append(ret, e)
		}
	}
	return ret
}
