package logger_test

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/nyaxt/otaru/logger"
)

func TestWriterLogger(t *testing.T) {
	var b bytes.Buffer
	l := logger.WriterLogger{&b}
	logger.Debugf(l, "foobar")

	expre := regexp.MustCompile("writerlogger_test.go:\\d+: foobar\n")
	if !expre.Match(b.Bytes()) {
		t.Errorf("Unexpected: %s", b.String())
	}
}
