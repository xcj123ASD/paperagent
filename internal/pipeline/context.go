package pipeline

import (
	"sync"
	"time"
)

type LogEntry struct {
	Time  string `json:"time"`
	Stage string `json:"stage"`
	Msg   string `json:"msg"`
}

type Context struct {
	Mock    bool
	Strict  bool
	StartAt time.Time

	mu  sync.Mutex
	Log []LogEntry
}

func NewContext(mock, strict bool) *Context {
	return &Context{Mock: mock, Strict: strict, StartAt: time.Now()}
}

func (c *Context) Logf(stage, msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Log = append(c.Log, LogEntry{Time: time.Now().Format("15:04:05"), Stage: stage, Msg: msg})
}

// Snapshot 返回日志的安全副本，供 Web 层轮询。
func (c *Context) Snapshot() []LogEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]LogEntry, len(c.Log))
	copy(out, c.Log)
	return out
}
