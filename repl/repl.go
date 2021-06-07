package repl

import (
	"bufio"
	"fmt"
	"io"

	"monkey/eval"
	"monkey/lexer"
	"monkey/object"
	"monkey/parser"
)

const PROMPT = ">>> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	env := object.NewEnvironment()

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
			for _, v := range p.Errors() {
				io.WriteString(out, v)
				io.WriteString(out, "\n")
			}
			continue
		}

		// io.WriteString(out, program.String())
		evaluated := eval.Eval(program, env)
		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
	}
}
