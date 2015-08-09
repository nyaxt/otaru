package logger

import (
	"bytes"
	"io"
	"time"

	"github.com/nyaxt/otaru/util"
)

type WriterLogger struct {
	W io.Writer
}

func (l WriterLogger) Log(lv Level, data map[string]interface{}) {
	t := data["time"].(time.Time)

	var b bytes.Buffer
	b.WriteString(t.Format("2006/01/02 15:04:05 "))
	b.WriteString(data["location"].(string))
	b.WriteString(": ")
	b.WriteString(data["log"].(string))
	b.WriteString("\n")
	l.W.Write(b.Bytes())

	if s, ok := l.W.(util.Syncer); ok {
		s.Sync()
	}
}

func (l WriterLogger) WillAccept(lv Level) bool { return true }
