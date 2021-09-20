package eval

import (
	"monkey/ast"
	"monkey/object"
)

func DefineMacros(program *ast.Program, env *object.Environment) {
	definitions := []int{}

	for i, stmt := range program.Statements {
		letStmt, ok := stmt.(*ast.LetStatement)
		if !ok {
			continue
		}
		macroLit, ok := letStmt.Value.(*ast.MacroLiteral)
		if !ok {
			continue
		}

		macro := &object.Macro{
			Params: macroLit.Params,
			Body:   macroLit.Body,
			Env:    env,
		}

		env.Set(letStmt.Name.String(), macro)

		definitions = append(definitions, i)
	}

	for i := len(definitions) - 1; i >= 0; i-- {
		defIdx := definitions[i]
		program.Statements = append(program.Statements[:defIdx], program.Statements[defIdx+1:]...)
	}
}

func ExpandMacros(program *ast.Program, env *object.Environment) ast.Node {
	return ast.Modify(program, func(node ast.Node) ast.Node {
		callExp, ok := node.(*ast.CallExpression)
		if !ok {
			return node
		}
		ident, ok := callExp.Function.(*ast.Identifier)
		if !ok {
			return node
		}
		obj, ok := env.Get(ident.Value)
		if !ok {
			return node
		}
		macro, ok := obj.(*object.Macro)
		if !ok {
			return node
		}

		args := []*object.Quote{}
		for _, arg := range callExp.Arguments {
			args = append(args, &object.Quote{Node: arg})
		}

		extendedEnv := object.NewEnclosedEnvironment(macro.Env)
		for i, param := range macro.Params {
			extendedEnv.Set(param.Value, args[i])
		}

		evaluated := Eval(macro.Body, extendedEnv)
		quote, ok := evaluated.(*object.Quote)
		if !ok {
			panic("we only support returning AST-nodes from macros")
		}

		return quote.Node
	})
}
