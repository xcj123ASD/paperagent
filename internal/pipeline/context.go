package pipeline

import "time"

type LogEntry struct {
	Time  string `json:"time"`
	Stage string `json:"stage"`
	Msg   string `json:"msg"`
}

type Context struct {
	Mock    bool
	Strict  bool
	Log     []LogEntry
	StartAt time.Time
}

func NewContext(mock, strict bool) *Context {
	return &Context{Mock: mock, Strict: strict, StartAt: time.Now()}
}

func (c *Context) Logf(stage, msg string) {
	c.Log = append(c.Log, LogEntry{Time: time.Now().Format("15:04:05"), Stage: stage, Msg: msg})
}
