package vm

import (
	"monkey/code"
	"monkey/object"
)

type Frame struct {
	fn      *object.CompiledFunction
	ip      int
	basePtr int
}

func NewFrame(fn *object.CompiledFunction, basePtr int) *Frame {
	return &Frame{fn: fn, ip: -1, basePtr: basePtr}
}

func (f *Frame) Instructions() code.Instructions {
	return f.fn.Instructions
}
