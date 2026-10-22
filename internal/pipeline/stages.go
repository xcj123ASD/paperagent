package pipeline

import "errors"

var ErrNotImplemented = errors.New("stage not implemented")

type Stage interface {
	Name() string
	Run(ctx *Context, input []byte) ([]byte, error)
}

var Registry = map[string]Stage{}

func Register(s Stage) { Registry[s.Name()] = s }

type Stub struct{ Name_ string }

func (s Stub) Name() string                       { return s.Name_ }
func (Stub) Run(*Context, []byte) ([]byte, error) { return nil, ErrNotImplemented }
