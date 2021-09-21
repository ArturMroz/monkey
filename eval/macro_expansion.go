package eval

import (
	"monkey/ast"
	"monkey/object"
)

// DefineMacros finds macro definitions, constructs macro out of them
// and adds them to Env, and finally removes the definitions from AST.
func DefineMacros(program *ast.Program, env *object.Environment) {
	for i := len(program.Statements) - 1; i >= 0; i-- {
		letStmt, ok := program.Statements[i].(*ast.LetStatement)
		if !ok {
			continue
		}
		macroLit, ok := letStmt.Value.(*ast.MacroLiteral)
		if !ok {
			// not a macro definition, bail
			continue
		}

		macro := &object.Macro{
			Params: macroLit.Params,
			Body:   macroLit.Body,
			Env:    env,
		}

		env.Set(letStmt.Name.String(), macro)

		// remove found macro definition from the AST
		program.Statements = append(program.Statements[:i], program.Statements[i+1:]...)
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

		// quote args
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
