package eval

import (
	"monkey/ast"
	"monkey/object"
)

func DefineMacros(program *ast.Program, env *object.Environment) {
	definitions := []int{}

	for i, stmt := range program.Statements {
		if isMacro(stmt) {
			// addMacro(stmt, env)

			letStmt := stmt.(*ast.LetStatement)
			macroLit := letStmt.Value.(*ast.MacroLiteral)

			macro := &object.Macro{
				Params: macroLit.Params,
				Body:   macroLit.Body,
				Env:    env,
			}

			env.Set(letStmt.Name.String(), macro)

			definitions = append(definitions, i)
		}
	}

	for i := len(definitions); i >= 0; i-- {
		defIdx := definitions[i]
		program.Statements = append(program.Statements[:defIdx], program.Statements[defIdx+1:]...)
	}
}

func isMacro(node ast.Statement) bool {
	letStmt, ok := node.(*ast.LetStatement)
	if !ok {
		return false
	}

	_, ok = letStmt.Value.(*ast.MacroLiteral)
	if !ok {
		return false
	}

	return true
}
