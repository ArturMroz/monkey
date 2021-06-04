package eval

import (
	"monkey/ast"
	"monkey/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node) object.Object {
	switch node := node.(type) {
	case *ast.Program:
		return evalStatements(node.Statements)

	case *ast.ExpressionStatement:
		return Eval(node.Expression)

	case *ast.PrefixExpression:
		right := Eval(node.Right)
		return evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		left := Eval(node.Left)
		right := Eval(node.Right)
		return evalInfixExpression(node.Operator, left, right)

	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBoolean(node.Value)
	}

	return nil
}

func evalStatements(stmts []ast.Statement) object.Object {
	var result object.Object
	for _, statement := range stmts {
		result = Eval(statement)
	}

	return result
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		// switch right := right.(type) {
		// case *object.Boolean:
		// 	return &object.Boolean{Value: !right.Value}
		// case *object.Integer:
		// 	val := right.Value != 0
		// 	return &object.Boolean{Value: !val}
		// }
		switch right {
		case TRUE:
			return FALSE
		case FALSE:
			return TRUE
		case NULL:
			return TRUE
		default:
			return FALSE
		}
	case "-":
		right, ok := right.(*object.Integer)
		if !ok {
			// "-" prefix operator supports only Integers
			return NULL
		}
		return &object.Integer{Value: -right.Value}
	default:
		return NULL
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
			return NULL
		}
	} else {
		switch operator {
		case "==":
			return nativeBoolToBoolean(left == right)
		case "!=":
			return nativeBoolToBoolean(left != right)
		default:
			return NULL
		}
	}
}

func nativeBoolToBoolean(value bool) *object.Boolean {
	if value {
		return TRUE
	}
	return FALSE
}
