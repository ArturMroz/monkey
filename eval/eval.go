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

func applyFunction(function object.Object, args []object.Object) object.Object {
	fn, ok := function.(*object.Function)
	if !ok {
		return newError("not a function: %s", fn.Type())
	}

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
