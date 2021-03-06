package vm

import (
	"fmt"

	"monkey/code"
	"monkey/compiler"
	"monkey/object"
)

const (
	StackSize   = 1024
	GlobalsSize = 65536
	MaxFrames   = 1024
)

var (
	True  = &object.Boolean{Value: true}
	False = &object.Boolean{Value: false}
	Null  = &object.Null{}
)

type VM struct {
	stack       []object.Object
	sp          int // Always points to the next value. Top of stack is stack[sp-1]
	constants   []object.Object
	globals     []object.Object
	frames      []*Frame
	framesIndex int
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	mainClosure := &object.Closure{Fn: mainFn}
	mainFrame := NewFrame(mainClosure, 0)
	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	return &VM{
		stack:       make([]object.Object, StackSize),
		globals:     make([]object.Object, GlobalsSize),
		constants:   bytecode.Constants,
		frames:      frames,
		framesIndex: 1,
	}
}

func NewWithGlobalsStore(bytecode *compiler.Bytecode, globals []object.Object) *VM {
	vm := New(bytecode)
	vm.globals = globals
	return vm
}

func (vm *VM) StackTop() object.Object {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) Run() error {
	var ip int
	var ins code.Instructions

	for vm.curFrame().ip < len(vm.curFrame().Instructions())-1 {
		vm.curFrame().ip++

		ip = vm.curFrame().ip
		ins = vm.curFrame().Instructions()

		switch op := code.Opcode(ins[ip]); op {
		case code.OpConstant:
			constIndex := code.ReadUint16(ins[ip+1:])
			vm.curFrame().ip += 2
			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}

		case code.OpSetGlobal:
			globalIdx := code.ReadUint16(ins[ip+1:])
			vm.curFrame().ip += 2
			vm.globals[globalIdx] = vm.pop()

		case code.OpGetGlobal:
			globalIdx := code.ReadUint16(ins[ip+1:])
			vm.curFrame().ip += 2
			err := vm.push(vm.globals[globalIdx])
			if err != nil {
				return err
			}

		case code.OpSetLocal:
			localIndex := code.ReadUint8(ins[ip+1:])
			vm.curFrame().ip += 1

			vm.stack[vm.curFrame().basePtr+int(localIndex)] = vm.pop()

		case code.OpGetLocal:
			localIndex := code.ReadUint8(ins[ip+1:])
			vm.curFrame().ip += 1

			err := vm.push(vm.stack[vm.curFrame().basePtr+int(localIndex)])
			if err != nil {
				return err
			}

		case code.OpGetFree:
			freeIndex := code.ReadUint8(ins[ip+1:])
			vm.curFrame().ip += 1
			currentClosure := vm.curFrame().cl

			err := vm.push(currentClosure.Free[freeIndex])
			if err != nil {
				return err
			}

		case code.OpCurrentClosure:
			curClosure := vm.curFrame().cl
			err := vm.push(curClosure)
			if err != nil {
				return err
			}

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				return err
			}

		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan:
			err := vm.executeComparison(op)
			if err != nil {
				return err
			}

		case code.OpBang:
			err := vm.executeBangOperator()
			if err != nil {
				return err
			}

		case code.OpMinus:
			err := vm.executeMinusOperator()
			if err != nil {
				return err
			}

		case code.OpPop:
			vm.pop()

		case code.OpJump:
			pos := int(code.ReadUint16(ins[ip+1:]))
			// as ip is increased with each loop, we need to offset it
			vm.curFrame().ip = pos - 1

		case code.OpJumpNotTruthy:
			pos := int(code.ReadUint16(ins[ip+1:]))
			vm.curFrame().ip += 2

			condition := vm.pop()
			if !isTruthy(condition) {
				vm.curFrame().ip = pos - 1
			}

		case code.OpCall:
			numArgs := int(code.ReadUint8(ins[ip+1:]))
			vm.curFrame().ip += 1

			fn := vm.stack[vm.sp-1-numArgs]
			switch callee := fn.(type) {
			case *object.Closure:
				if numArgs != callee.Fn.NumParams {
					return fmt.Errorf("wrong number of arguments: want=%d, got=%d", callee.Fn.NumParams, numArgs)
				}

				frame := NewFrame(callee, vm.sp-numArgs)
				vm.pushFrame(frame)
				vm.sp = frame.basePtr + callee.Fn.NumLocals

			case *object.Builtin:
				args := vm.stack[vm.sp-numArgs : vm.sp]
				result := callee.Fn(args...)
				vm.sp -= numArgs + 1

				if result != nil {
					vm.push(result)
				} else {
					vm.push(Null)
				}

			default:
				return fmt.Errorf("calling non-closure and non-built-in")
			}

		case code.OpClosure:
			constIdx := code.ReadUint16(ins[ip+1:])
			numFree := int(code.ReadUint8(ins[ip+3:]))
			vm.curFrame().ip += 3

			constant := vm.constants[constIdx]
			fn, ok := constant.(*object.CompiledFunction)
			if !ok {
				return fmt.Errorf("not a function: %+v", constant)
			}

			free := make([]object.Object, numFree)
			// for i := 0; i < numFree; i++ {
			// 	free[i] = vm.stack[vm.sp-numFree+i]
			// }
			copy(free, vm.stack[vm.sp-numFree:]) // copy free variables for closure from the stack
			vm.sp -= numFree                     // clean the stack

			closure := &object.Closure{Fn: fn, Free: free}
			err := vm.push(closure)
			if err != nil {
				return err
			}

		case code.OpReturnValue:
			returnValue := vm.pop()

			frame := vm.popFrame()
			vm.sp = frame.basePtr - 1

			err := vm.push(returnValue)
			if err != nil {
				return err
			}

		case code.OpReturn:
			frame := vm.popFrame()
			vm.sp = frame.basePtr - 1

			err := vm.push(Null)
			if err != nil {
				return err
			}

		case code.OpGetBuiltin:
			builtinIndex := code.ReadUint8(ins[ip+1:])
			vm.curFrame().ip += 1

			definition := object.Builtins[builtinIndex]

			err := vm.push(definition.Builtin)
			if err != nil {
				return err
			}

		case code.OpTrue:
			if err := vm.push(True); err != nil {
				return err
			}

		case code.OpFalse:
			if err := vm.push(False); err != nil {
				return err
			}

		case code.OpNull:
			if err := vm.push(Null); err != nil {
				return err
			}

		case code.OpArray:
			numElements := int(code.ReadUint16(ins[ip+1:]))
			vm.curFrame().ip += 2

			start, end := vm.sp-numElements, vm.sp
			elems := make([]object.Object, end-start)
			copy(elems, vm.stack[start:end])
			array := &object.Array{Elements: elems}

			vm.sp -= numElements

			if err := vm.push(array); err != nil {
				return err
			}

		case code.OpHash:
			numElements := int(code.ReadUint16(ins[ip+1:]))
			vm.curFrame().ip += 2

			hash, err := vm.buildHash(vm.sp-numElements, vm.sp)
			if err != nil {
				return err
			}

			vm.sp -= numElements

			if err := vm.push(hash); err != nil {
				return err
			}

		case code.OpIndex:
			index := vm.pop()
			left := vm.pop()
			if err := vm.executeIndexExpression(left, index); err != nil {
				return err
			}
		}
	}

	return nil
}

func (vm *VM) executeIndexExpression(left, index object.Object) error {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeArrayIndex(left, index)
	case left.Type() == object.HASH_OBJ:
		return vm.executeHashIndex(left, index)
	default:
		return fmt.Errorf("index operator not supported: %s", left.Type())
	}
}

func (vm *VM) executeArrayIndex(array, index object.Object) error {
	arrayObject := array.(*object.Array)
	i := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if i < 0 || i > max {
		return vm.push(Null)
	}

	return vm.push(arrayObject.Elements[i])
}

func (vm *VM) executeHashIndex(hash, index object.Object) error {
	hashObject := hash.(*object.Hash)
	key, ok := index.(object.Hashable)
	if !ok {
		return fmt.Errorf("unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObject.Pairs[key.HashKey()]
	if !ok {
		return vm.push(Null)
	}
	return vm.push(pair.Value)
}

func (vm *VM) buildHash(start, end int) (*object.Hash, error) {
	hashedPairs := map[object.HashKey]object.HashPair{}
	for i := start; i < end; i += 2 {
		key := vm.stack[i]
		value := vm.stack[i+1]
		pair := object.HashPair{Key: key, Value: value}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return nil, fmt.Errorf("unusable as hash key: %s", key.Type())
		}

		hashedPairs[hashKey.HashKey()] = pair
	}

	return &object.Hash{Pairs: hashedPairs}, nil
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	if left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ {
		leftValue := left.(*object.Integer).Value
		rightValue := right.(*object.Integer).Value

		switch op {
		case code.OpEqual:
			return vm.push(nativeBoolToBooleanObject(rightValue == leftValue))
		case code.OpNotEqual:
			return vm.push(nativeBoolToBooleanObject(rightValue != leftValue))
		case code.OpGreaterThan:
			return vm.push(nativeBoolToBooleanObject(leftValue > rightValue))
		default:
			return fmt.Errorf("unknown operator: %d", op)
		}
	}

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(right == left))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(right != left))
	default:
		return fmt.Errorf("unknown operator: %d (%s %s)", op, left.Type(), right.Type())
	}
}

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()
	leftType := left.Type()
	rightType := right.Type()

	switch {
	case leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ:
		leftVal := left.(*object.Integer).Value
		rightVal := right.(*object.Integer).Value
		var result int64
		switch op {
		case code.OpAdd:
			result = leftVal + rightVal
		case code.OpSub:
			result = leftVal - rightVal
		case code.OpMul:
			result = leftVal * rightVal
		case code.OpDiv:
			result = leftVal / rightVal
		default:
			return fmt.Errorf("unknown integer operator: %d", op)
		}
		return vm.push(&object.Integer{Value: result})

	case leftType == object.STRING_OBJ && rightType == object.STRING_OBJ:
		if op != code.OpAdd {
			return fmt.Errorf("unknown string operator: %d", op)
		}
		leftVal := left.(*object.String).Value
		rightVal := right.(*object.String).Value
		return vm.push(&object.String{Value: leftVal + rightVal})

	default:
		return fmt.Errorf("unsupported types for binary operation: %s %s", leftType, rightType)
	}
}

func (vm *VM) executeBangOperator() error {
	operand := vm.pop()
	switch operand {
	case True:
		return vm.push(False)
	case False:
		return vm.push(True)
	case Null:
		return vm.push(True)
	default:
		return vm.push(False)
	}
}

func (vm *VM) executeMinusOperator() error {
	operand := vm.pop()

	if operand.Type() != object.INTEGER_OBJ {
		return fmt.Errorf("unsupported type for negation: %s", operand.Type())
	}

	value := operand.(*object.Integer).Value
	return vm.push(&object.Integer{Value: -value})
}

func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) curFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return True
	}
	return False
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Boolean:
		return obj.Value

	case *object.Null:
		return false

	default:
		return true
	}
}
