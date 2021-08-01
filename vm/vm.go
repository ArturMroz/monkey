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
)

var (
	True  = &object.Boolean{Value: true}
	False = &object.Boolean{Value: false}
	Null  = &object.Null{}
)

type VM struct {
	stack        []object.Object
	sp           int // Always points to the next value. Top of stack is stack[sp-1]
	instructions code.Instructions
	constants    []object.Object
	globals      []object.Object
}

func New(bytecode *compiler.Bytecode) *VM {
	return &VM{
		stack:        make([]object.Object, StackSize),
		instructions: bytecode.Instructions,
		constants:    bytecode.Constants,
		globals:      make([]object.Object, GlobalsSize),
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
	for ip := 0; ip < len(vm.instructions); ip++ {
		switch op := code.Opcode(vm.instructions[ip]); op {
		case code.OpConstant:
			constIndex := code.ReadUint16(vm.instructions[ip+1:])
			ip += 2
			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}

		case code.OpSetGlobal:
			globalIdx := code.ReadUint16(vm.instructions[ip+1:])
			ip += 2
			vm.globals[globalIdx] = vm.pop()

		case code.OpGetGlobal:
			globalIdx := code.ReadUint16(vm.instructions[ip+1:])
			ip += 2
			err := vm.push(vm.globals[globalIdx])
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
			pos := int(code.ReadUint16(vm.instructions[ip+1:]))
			// as ip is increased with each loop, we need to offset it
			ip = pos - 1

		case code.OpJumpNotTruthy:
			pos := int(code.ReadUint16(vm.instructions[ip+1:]))
			ip += 2

			condition := vm.pop()
			if !isTruthy(condition) {
				ip = pos - 1
			}

		case code.OpTrue:
			err := vm.push(True)
			if err != nil {
				return err
			}

		case code.OpFalse:
			err := vm.push(False)
			if err != nil {
				return err
			}

		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				return err
			}

		}
	}

	return nil
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

	if leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ {
		leftValue := left.(*object.Integer).Value
		rightValue := right.(*object.Integer).Value
		var result int64
		switch op {
		case code.OpAdd:
			result = leftValue + rightValue
		case code.OpSub:
			result = leftValue - rightValue
		case code.OpMul:
			result = leftValue * rightValue
		case code.OpDiv:
			result = leftValue / rightValue
		default:
			return fmt.Errorf("unknown integer operator: %d", op)
		}

		return vm.push(&object.Integer{Value: result})
	}

	return fmt.Errorf("unsupported types for binary operation: %s %s", leftType, rightType)
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
