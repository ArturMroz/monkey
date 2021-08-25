package vm

import (
	"monkey/code"
	"monkey/object"
)

type Frame struct {
	cl      *object.Closure
	ip      int
	basePtr int
}

func NewFrame(cl *object.Closure, basePtr int) *Frame {
	return &Frame{cl: cl, ip: -1, basePtr: basePtr}
}

func (f *Frame) Instructions() code.Instructions {
	return f.cl.Fn.Instructions
}
