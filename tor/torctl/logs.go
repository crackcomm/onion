package torctl

import (
	"bytes"

	"github.com/golang/glog"
)

// logReader - Reads TOR log and sends a signal on Done() channel
// when reads `Bootstrapped 100%: Done`.
type logReader struct {
	quiet  bool
	done   bool
	doneCh chan interface{}
}

// newLogReader - New TOR Log reader.
func newLogReader(quiet bool) *logReader {
	return &logReader{
		quiet:  quiet,
		doneCh: make(chan interface{}, 1),
	}
}

var torLogNotice = []byte(`[notice]`)
var torDone = []byte(`100%`)

func (self *logReader) Done() <-chan interface{} {
	return self.doneCh
}

func (self *logReader) Write(b []byte) (int, error) {
	i := bytes.Index(b, torLogNotice)
	if i < 0 {
		return len(b), nil
	}

	start := i + len(torLogNotice) + 1
	if start > len(b) {
		return len(b), nil
	}

	if !self.done && bytes.Contains(b, torDone) {
		self.done = true
		self.doneCh <- true
		close(self.doneCh)
	}

	if !self.quiet {
		glog.Infof("[tor] %s", b[start:len(b)-2])
	}

	return len(b), nil
}
