package eval

import (
	"fmt"

	"monkey/ast"
	"monkey/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

var builtins = map[string]*object.Builtin{
	"len": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			switch arg := args[0].(type) {
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
			default:
				return newError("argument to `len` not supported, got %s",
					args[0].Type())
			}
		},
	},
	"first": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("argument to `first` must be ARRAY, got %s", args[0].Type())
			}
			if len(arr.Elements) > 0 {
				return arr.Elements[0]
			}
			return NULL
		},
	},
	"last": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("argument to `last` must be ARRAY, got %s", args[0].Type())
			}
			if len(arr.Elements) > 0 {
				return arr.Elements[len(arr.Elements)-1]
			}
			return NULL
		},
	},
	"rest": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("argument to `rest` must be ARRAY, got %s", args[0].Type())
			}
			length := len(arr.Elements)
			if length > 0 {
				newElements := make([]object.Object, 0, length-1)
				copy(newElements, arr.Elements[1:length])
				return &object.Array{Elements: newElements}
			}
			return NULL
		},
	},
	"push": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("argument to `push` must be ARRAY, got %s", args[0].Type())
			}

			length := len(arr.Elements)
			newElements := make([]object.Object, 0, length+1)
			copy(newElements, arr.Elements)
			newElements = append(newElements, args[1])
			return &object.Array{Elements: newElements}
		},
	},
	"puts": {
		Fn: func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}
			return NULL
		},
	},
}

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalProgram(node.Statements, env)

	case *ast.LetStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)

	case *ast.Identifier:
		if val, ok := env.Get(node.Value); ok {
			return val
		}
		if builtin, ok := builtins[node.Value]; ok {
			return builtin
		}
		return newError("identifier not found: " + node.Value)

	case *ast.FunctionLiteral:
		return &object.Function{
			Params: node.Params,
			Body:   node.Body,
			Env:    env,
		}

	case *ast.CallExpression:
		fn := Eval(node.Function, env)
		if isError(fn) {
			return fn
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(fn, args)

	case *ast.ArrayLiteral:
		elts := evalExpressions(node.Elements, env)
		if len(elts) == 1 && isError(elts[0]) {
			return elts[0]
		}
		return &object.Array{Elements: elts}

	case *ast.HashLiteral:
		pairs := map[object.HashKey]object.HashPair{}
		for k, v := range node.Pairs {
			key := Eval(k, env)
			if isError(key) {
				return key
			}
			hashKey, ok := key.(object.Hashable)
			if !ok {
				return newError("unusable as hash key: %s", key.Type())
			}
			value := Eval(v, env)
			if isError(value) {
				return value
			}
			hashed := hashKey.HashKey()
			pairs[hashed] = object.HashPair{Key: key, Value: value}
		}
		return &object.Hash{Pairs: pairs}

	case *ast.BlockStatement:
		var result object.Object
		for _, statement := range node.Statements {
			result = Eval(statement, env)
			if result != nil && (result.Type() == object.RETURN_VALUE_OBJ || result.Type() == object.ERROR_OBJ) {
				return result
			}
		}
		return result

	case *ast.IfExpression:
		return evalIfExpression(node, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}

		switch {
		case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
			array := left.(*object.Array)
			idx := index.(*object.Integer).Value
			if idx < 0 || idx > int64(len(array.Elements)-1) {
				return NULL
			}
			return array.Elements[idx]
		case left.Type() == object.HASH_OBJ:
			hash := left.(*object.Hash)
			key, ok := index.(object.Hashable)
			if !ok {
				return newError("unusable as hash key: %s", index.Type())
			}
			pair, ok := hash.Pairs[key.HashKey()]
			if !ok {
				return NULL
			}
			return pair.Value
		default:
			return newError("index operator not supported: %s", left.Type())
		}

	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBoolean(node.Value)
	}

	return nil
}

func evalProgram(stmts []ast.Statement, env *object.Environment) object.Object {
	var result object.Object
	for _, statement := range stmts {
		result = Eval(statement, env)
		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}
	return result
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return nativeBoolToBoolean(!isTruthy(right))
	case "-":
		if right, ok := right.(*object.Integer); ok {
			return &object.Integer{Value: -right.Value}
		}
		return newError("infix operator '-' supports only integers, got %s", right.Type())
	default:
		return newError("unknown operator: %s %s", operator, right.Type())
	}
}

func evalInfixExpression(operator string, left, right object.Object) object.Object {
	if left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ {
		leftVal := left.(*object.Integer).Value
		rightVal := right.(*object.Integer).Value
		switch operator {
		case "+":
			return &object.Integer{Value: leftVal + rightVal}
		case "-":
			return &object.Integer{Value: leftVal - rightVal}
		case "*":
			return &object.Integer{Value: leftVal * rightVal}
		case "/":
			return &object.Integer{Value: leftVal / rightVal}
		case "<":
			return nativeBoolToBoolean(leftVal < rightVal)
		case ">":
			return nativeBoolToBoolean(leftVal > rightVal)
		case "==":
			return nativeBoolToBoolean(leftVal == rightVal)
		case "!=":
			return nativeBoolToBoolean(leftVal != rightVal)
		default:
			return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
		}
	} else if left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ {
		switch operator {
		case "==":
			return nativeBoolToBoolean(left == right)
		case "!=":
			return nativeBoolToBoolean(left != right)
		default:
			return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
		}
	} else if left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ {
		if operator != "+" {
			return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
		}
		leftVal := left.(*object.String).Value
		rightVal := right.(*object.String).Value
		return &object.String{Value: leftVal + rightVal}
	} else {
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object
	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	switch fn := fn.(type) {
	case *object.Function:
		extendedEnv := object.NewEnclosedEnvironment(fn.Env)
		for i := range fn.Params {
			extendedEnv.Set(fn.Params[i].Value, args[i])
		}
		evaluated := Eval(fn.Body, extendedEnv)
		if returnValue, ok := evaluated.(*object.ReturnValue); ok {
			// unwrap ReturnValue so it doesn't bubble up the chain
			return returnValue.Value
		}
		return evaluated

	case *object.Builtin:
		return fn.Fn(args...)

	default:
		return newError("not a function: %s", fn.Type())
	}
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func nativeBoolToBoolean(value bool) *object.Boolean {
	if value {
		return TRUE
	}
	return FALSE
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case TRUE:
		return true
	case FALSE:
		return false
	case NULL:
		return false
	default:
		return true
	}
}
