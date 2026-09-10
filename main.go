package main

import (
	"fmt"
	"os"
)

const usage = `gopy - Python subset interpreter in Go

Usage:
  gopy                start interactive REPL
  gopy <file.py>      interpret a Python source file

Supported Python subset:
  - Literals: int, float, str, bool (True/False), None
  - Operators: + - * / %, == != < > <= >=, and or not, unary -
  - Statements: assignment (=), if/elif/else, while, for-in, def, return, break, continue, pass
  - Expressions: function call, list literal [], indexing arr[i], ternary 'a if c else b'
  - Builtins: print, len, range, str, int, float, type

Examples:
  gopy examples/fib.py
  gopy examples/fizzbuzz.py

Debug:
  GOPY_TOKENS=1 gopy <file>   dump token stream
  GOPY_AST=1 gopy <file>      dump AST
`

func main() {
	if os.Getenv("GOPY_TOKENS") != "" {
		runTokensDump(os.Args[1:])
		return
	}
	if os.Getenv("GOPY_AST") != "" {
		runASTDump(os.Args[1:])
		return
	}

	if len(os.Args) < 2 {
		StartREPL(os.Stdin, os.Stdout)
		return
	}
	arg := os.Args[1]
	if arg == "-h" || arg == "--help" {
		fmt.Print(usage)
		return
	}
	if err := RunFile(arg); err != nil {
		os.Exit(1)
	}
}

func runTokensDump(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: GOPY_TOKENS=1 gopy <file>")
		os.Exit(1)
	}
	data, _ := os.ReadFile(args[0])
	l := NewLexer(string(data))
	for {
		tok := l.NextToken()
		fmt.Printf("%-10s %-12q  L%d:C%d  ind=%d\n", tok.Type, tok.Literal, tok.Line, tok.Column, tok.Indent)
		if tok.Type == EOF {
			break
		}
	}
}

func runASTDump(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: GOPY_AST=1 gopy <file>")
		os.Exit(1)
	}
	data, _ := os.ReadFile(args[0])
	l := NewLexer(string(data))
	p := NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintln(os.Stderr, "parse error:", e)
		}
		os.Exit(1)
	}
	fmt.Println(program.String())
}