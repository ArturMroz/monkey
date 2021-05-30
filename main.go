package main

import (
	"fmt"
	"os"

	"monkey/repl"
)

func main() {
	fmt.Println("Welcome to Monkey REPL")
	repl.Start(os.Stdin, os.Stdout)
}
