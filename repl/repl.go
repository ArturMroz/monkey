package repl

import (
	"bufio"
	"fmt"
	"io"

	"monkey/compiler"
	"monkey/lexer"
	"monkey/parser"
	"monkey/vm"
)

const PROMPT = ">>> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	// env := object.NewEnvironment()

	for {
		fmt.Fprintf(out, PROMPT)
		if !scanner.Scan() {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			io.WriteString(out, "Parsing failed. Errors:\n")
			for _, v := range p.Errors() {
				io.WriteString(out, v)
				io.WriteString(out, "\n")
			}
			continue
		}

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			fmt.Fprintf(out, "Compilation failed:\n %s\n", err)
			continue
		}

		machine := vm.New(comp.Bytecode())
		err = machine.Run()
		if err != nil {
			fmt.Fprintf(out, "Executing bytecode failed:\n %s\n", err)
			continue
		}

		lastPopped := machine.LastPoppedStackElem()
		io.WriteString(out, lastPopped.Inspect())
		io.WriteString(out, "\n")

		// evaluated := eval.Eval(program, env)
		// if evaluated != nil {
		// 	io.WriteString(out, evaluated.Inspect())
		// 	io.WriteString(out, "\n")
		// }
	}
}
