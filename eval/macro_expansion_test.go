package eval

import (
	"testing"

	"monkey/ast"
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
)

func TestDefineMacros(t *testing.T) {
	input := `
    let number = 1;
    let function = fn(x, y) { x + y };
    let mymacro = macro(x, y) { x + y; };
    `

	env := object.NewEnvironment()
	program := testParseProgram(input)

	DefineMacros(program, env)

	if len(program.Statements) != 2 {
		t.Fatalf("Wrong number of statements. got=%d", len(program.Statements))
	}

	_, ok := env.Get("number")
	if ok {
		t.Fatalf("number should not be defined")
	}
	_, ok = env.Get("function")
	if ok {
		t.Fatalf("function should not be defined")
	}

	obj, ok := env.Get("mymacro")
	if !ok {
		t.Fatalf("macro not in environment.")
	}

	macro, ok := obj.(*object.Macro)
	if !ok {
		t.Fatalf("object is not Macro. got=%T (%+v)", obj, obj)
	}

	if len(macro.Params) != 2 {
		t.Fatalf("Wrong number of macro parameters. got=%d", len(macro.Params))
	}

	if macro.Params[0].String() != "x" {
		t.Fatalf("parameter is not 'x'. got=%q", macro.Params[0])
	}
	if macro.Params[1].String() != "y" {
		t.Fatalf("parameter is not 'y'. got=%q", macro.Params[1])
	}

	expectedBody := "(x + y)"

	if macro.Body.String() != expectedBody {
		t.Fatalf("body is not %q. got=%q", expectedBody, macro.Body.String())
	}
}

func testParseProgram(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}
