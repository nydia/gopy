package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// StartREPL 启动交互式解释器（多行累积，空行触发执行）。
// 设计：把每段累积内容当做一个完整程序来处理，遇到 EOF 也会执行。
func StartREPL(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	env := NewEnvironment()

	fmt.Fprintln(out, "gopy 0.1 - Python subset interpreter in Go")
	fmt.Fprintln(out, "Type a statement then press Enter on an empty line to execute.")

	var buf strings.Builder
	// 跟踪当前缩进级别，用于智能判断"是否需要继续读"
	openIndents := 0

	for {
		if openIndents > 0 {
			fmt.Fprint(out, strings.Repeat(" ", openIndents))
		}
		fmt.Fprint(out, ">>> ")
		if !scanner.Scan() {
			// EOF：把残余内容也跑一下
			if buf.Len() > 0 {
				runAndPrint(buf.String(), env, out)
				buf.Reset()
			}
			fmt.Fprintln(out)
			return
		}
		line := scanner.Text()
		trimmed := strings.TrimLeft(line, " \t")

		// 空行（只有空白）：触发执行
		if trimmed == "" {
			if buf.Len() > 0 {
				runAndPrint(buf.String(), env, out)
				buf.Reset()
				openIndents = 0
			}
			continue
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)

		// 重新扫描整个 buffer 来更新缩进级别（轻量级）
		openIndents = countOpenIndents(buf.String())
	}
}

// countOpenIndents 用 lexer 数当前未关闭的 INDENT 数。
func countOpenIndents(src string) int {
	l := NewLexer(src)
	depth := 0
	for {
		tok := l.NextToken()
		switch tok.Type {
		case INDENT:
			depth++
		case DEDENT:
			if depth > 0 {
				depth--
			}
		case EOF:
			return depth
		}
	}
}

// runAndPrint 把源代码跑一次，把最后一条 ExpressionStatement 的值打印出来。
func runAndPrint(src string, env *Environment, out io.Writer) {
	l := NewLexer(src)
	p := NewParser(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintf(out, "  parse error: %s\n", e)
		}
		return
	}
	result := Eval(program, env)
	if IsError(result) {
		fmt.Fprintf(out, "  runtime error: %s\n", result.(*Error).Message)
		return
	}
	// 只在"纯表达式程序"时打印结果：有语句（比如 print、let）时不重复打印。
	if shouldPrintResult(program) {
		fmt.Fprintf(out, "%s\n", result.Inspect())
	}
}

// shouldPrintResult 当 program 只包含一条 ExpressionStatement 时返回 true。
// 这是 Python REPL 的行为：>>> 1+1 打印 2；>>> x = 1 不打印。
func shouldPrintResult(program *Program) bool {
	if len(program.Statements) != 1 {
		return false
	}
	_, ok := program.Statements[0].(*ExpressionStatement)
	return ok
}

// RunFile 解释执行一个 .py 文件。
func RunFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	env := NewEnvironment()
	// 把内建 print 等都放到全局作用域
	// （evalIdentifier 已经处理了内建查找，无需额外注册）

	l := NewLexer(string(data))
	p := NewParser(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintf(os.Stderr, "parse error: %s\n", e)
		}
		return fmt.Errorf("parse failed")
	}
	result := Eval(program, env)
	if IsError(result) {
		// 未捕获异常按 CPython 风格显示类型名：KeyError: missing
		errObj := result.(*Error)
		msg := errObj.Message
		if errObj.Kind != "" {
			msg = errObj.Kind + ": " + msg
		}
		fmt.Fprintf(os.Stderr, "runtime error: %s\n", msg)
		return fmt.Errorf("runtime failed")
	}
	return nil
}