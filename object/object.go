package object

import (
	"bytes"
	"fmt"
	"hash/fnv"

	"monkey/ast"
	"monkey/code"
)

type ObjectType string

const (
	INTEGER_OBJ           ObjectType = "INTEGER"
	BOOLEAN_OBJ           ObjectType = "BOOLEAN"
	STRING_OBJ            ObjectType = "STRING"
	NULL_OBJ              ObjectType = "NULL"
	RETURN_VALUE_OBJ      ObjectType = "RETURN_VALUE"
	ERROR_OBJ             ObjectType = "ERROR"
	FUNCTION_OBJ          ObjectType = "FUNCTION"
	COMPILED_FUNCTION_OBJ ObjectType = "COMPILED_FUNCTION"
	CLOSURE_OBJ           ObjectType = "CLOSURE"
	BUILTIN_OBJ           ObjectType = "BUILTIN"
	ARRAY_OBJ             ObjectType = "ARRAY"
	HASH_OBJ              ObjectType = "HASH"
	QUOTE_OBJ             ObjectType = "QUOTE"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

type Function struct {
	Params []*ast.Identifier
	Body   *ast.BlockStatement
	Env    *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	var out bytes.Buffer
	out.WriteString("fn(")
	for i, p := range f.Params {
		out.WriteString(p.String())
		if i < len(f.Params)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString(") {\n")
	out.WriteString(f.Body.String())
	out.WriteString("\n}")
	return out.String()
}

type CompiledFunction struct {
	Instructions code.Instructions
	NumLocals    int
	NumParams    int
}

func (cf *CompiledFunction) Type() ObjectType { return COMPILED_FUNCTION_OBJ }
func (cf *CompiledFunction) Inspect() string  { return fmt.Sprintf("CompiledFunction[%p]", cf) }

type BuiltinFunction func(args ...Object) Object

type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

type Closure struct {
	Fn   *CompiledFunction
	Free []Object
}

func (c *Closure) Type() ObjectType { return CLOSURE_OBJ }
func (c *Closure) Inspect() string  { return fmt.Sprintf("Closure[%p]", c) }

type Array struct {
	Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	var out bytes.Buffer
	out.WriteString("[")
	for i, e := range a.Elements {
		out.WriteString(e.Inspect())
		if i < len(a.Elements)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString("]")
	return out.String()
}

type HashKey struct {
	Type  ObjectType
	Value uint64
}

func (b *Boolean) HashKey() HashKey {
	var value uint64
	if b.Value {
		value = 1
	} else {
		value = 0
	}
	return HashKey{Type: b.Type(), Value: value}
}

func (i *Integer) HashKey() HashKey {
	return HashKey{Type: i.Type(), Value: uint64(i.Value)}
}

func (s *String) HashKey() HashKey {
	h := fnv.New64a()
	h.Write([]byte(s.Value))
	return HashKey{Type: s.Type(), Value: h.Sum64()}
}

type Hashable interface {
	HashKey() HashKey
}

type HashPair struct {
	Key   Object
	Value Object
}

type Hash struct {
	Pairs map[HashKey]HashPair
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string {
	var out bytes.Buffer
	// pairs := []string{}
	out.WriteString("{")
	for _, pair := range h.Pairs {
		out.WriteString(fmt.Sprintf("%s: %s", pair.Key.Inspect(), pair.Value.Inspect()))
		// TODO remove last comma
		// if i < len(h.Pairs)-1 {
		// 	out.WriteString(", ")
		// }
	}
	out.WriteString("}")
	return out.String()
}

type Quote struct {
	Node ast.Node
}

func (q *Quote) Type() ObjectType { return QUOTE_OBJ }
func (q *Quote) Inspect() string  { return "QUOTE(" + q.Node.String() + ")" }
