package logger

import (
	"bytes"
	"io"
	"time"
)

type WriterLogger struct {
	W io.Writer
}

type syncer interface { // avoid using github.com/nyaxt/otaru/util.Syncer for refcycle
	Sync() error
}

func (l WriterLogger) Log(lv Level, data map[string]interface{}) {
	var b bytes.Buffer
	t := data["time"].(time.Time)
	b.WriteString(t.Format("2006/01/02 15:04:05 "))
	b.WriteRune(lv.Rune())
	b.WriteRune(' ')
	if c, ok := data["category"]; ok {
		b.WriteString("[")
		b.WriteString(c.(string))
		b.WriteString("] ")
	}

	b.WriteString(data["location"].(string))
	b.WriteString(": ")
	b.WriteString(data["log"].(string))
	b.WriteString("\n")
	l.W.Write(b.Bytes())

	if s, ok := l.W.(syncer); ok {
		s.Sync()
	}
}

func (l WriterLogger) WillAccept(lv Level) bool { return true }
