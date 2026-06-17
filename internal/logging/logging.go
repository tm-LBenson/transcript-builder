package logging

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type RunLogger struct {
	mu   sync.Mutex
	file *os.File
}

func New(path string) (*RunLogger, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &RunLogger{file: file}, nil
}

func (l *RunLogger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *RunLogger) Printf(format string, args ...any) {
	if l == nil || l.file == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	prefix := time.Now().Format(time.RFC3339)
	fmt.Fprintf(l.file, "%s %s\n", prefix, strings.TrimRight(fmt.Sprintf(format, args...), "\r\n"))
}

func (l *RunLogger) Section(title string) {
	l.Printf("== %s ==", title)
}

func (l *RunLogger) Output(label string, data []byte, allowBody bool) {
	if len(data) == 0 {
		l.Printf("%s: <empty>", label)
		return
	}
	if allowBody {
		l.Printf("%s:\n%s", label, strings.TrimRight(string(data), "\r\n"))
		return
	}
	l.Printf("%s: captured %d bytes (body withheld)", label, len(data))
}
