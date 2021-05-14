package main

import (
	"fmt"
	"monkey/repl"
	"os"
)

func main() {
	fmt.Printf("Monkey REPL")
	repl.Start(os.Stdin, os.Stdout)
}
