package main

import (
	"fmt"
	"os"
	"strings"
)

// Lexer 把源代码字符串切分成 token 流。
// Python 是缩进敏感语言，所以 lexer 还要负责产出 INDENT / DEDENT token。
type Lexer struct {
	input       string
	position    int // 当前字符位置（指向 ch）
	readPosition int // 下一个字符位置
	ch          byte // 当前字符
	line        int // 当前行号（1-based）
	col         int // 当前列号（1-based）

	// 缩进栈：当前已开启的缩进层级（用空格数表示）。
	// 顶层永远在 0 级。INDENT 入栈；DEDENT 弹栈。
	indentStack []int

	// atLineStart 表示下一次 lex 必须先处理行首（缩进判断）。
	atLineStart bool

	// tokenQueue 用来暂存多出来的 token（多级 DEDENT）。
	tokenQueue []Token
}

// NewLexer 用源代码字符串构造一个 Lexer。
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:       input,
		indentStack: []int{0},
		atLineStart: true,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	if l.ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken 返回下一个 token。
func (l *Lexer) NextToken() Token {
	// 1. 先消耗队列中已排队的 token（DEDENT）
	if len(l.tokenQueue) > 0 {
		t := l.tokenQueue[0]
		l.tokenQueue = l.tokenQueue[1:]
		return t
	}

	// 2. 行首：先处理缩进
	if l.atLineStart {
		return l.handleLineStart()
	}

	// 3. 行内空白
	l.skipInlineWhitespace()

	// 4. 注释跳到行尾
	if l.ch == '#' {
		l.skipToNextLine()
	}

	// 5. 文件结束
	if l.ch == 0 {
		return l.handleEOF()
	}

	// 6. 换行（语句分隔）。\r 单独出现或成对 \r\n 都当作换行，
	// 兼容 Windows 的 CRLF 与旧 Mac 的 CR。
	if l.ch == '\n' || l.ch == '\r' {
		return l.consumeNewline()
	}

	// 7. 其他词法单元
	return l.lexOneToken()
}

// ----------------------------------------------------------------------------
// 行首：缩进 / 反缩进
// ----------------------------------------------------------------------------

func (l *Lexer) handleLineStart() Token {
	for {
		l.atLineStart = false

		indent, onlyBlank := l.measureIndentAtLineStart()
		if l.ch == 0 {
			// 文件末尾：交由 handleEOF 处理
			return l.handleEOF()
		}
		if onlyBlank {
			// 空行 / 纯注释行 → 继续看下一行
			l.atLineStart = true
			continue
		}

		cur := l.indentStack[len(l.indentStack)-1]
		switch {
		case indent > cur:
			// 进入新一级缩进
			l.indentStack = append(l.indentStack, indent)
			return l.newToken(INDENT, "")
		case indent < cur:
			// 退出一级或多级缩进
			var dedents []Token
			for len(l.indentStack) > 1 && l.indentStack[len(l.indentStack)-1] > indent {
				l.indentStack = l.indentStack[:len(l.indentStack)-1]
				dedents = append(dedents, l.newToken(DEDENT, ""))
			}
			if l.indentStack[len(l.indentStack)-1] != indent {
				return l.newToken(ILLEGAL, "inconsistent indentation")
			}
			first := dedents[0]
			for _, d := range dedents[1:] {
				l.tokenQueue = append(l.tokenQueue, d)
			}
			return first
		default:
			// 缩进不变
			return l.lexOneToken()
		}
	}
}

// measureIndentAtLineStart 测量当前光标位置处行首的空格数。
// 返回 (缩进空格数, 是否整行都是空白或注释)。
// 读完后光标停在第一个非空白字符上。
func (l *Lexer) measureIndentAtLineStart() (int, bool) {
	if os.Getenv("GOPY_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[measureIndent] start ch=%q (0x%x) at L%d:C%d\n", l.ch, l.ch, l.line, l.col)
	}
	count := 0
	for l.ch == ' ' || l.ch == '\t' {
		count++
		l.readChar()
	}
	if l.ch == '\n' || l.ch == '\r' || l.ch == 0 {
		// 空行（可能以 \n、\r 或 \r\n 结尾）：消费掉换行序列，继续看下一行
		if l.ch == '\r' {
			l.readChar()
			if l.ch == '\n' {
				l.readChar()
			}
		} else if l.ch == '\n' {
			l.readChar()
		}
		return 0, true
	}
	if l.ch == '#' {
		l.skipToNextLine()
		// skipToNextLine 停在 \n 或 EOF；再前进一步跳过 \n
		if l.ch == '\n' {
			l.readChar()
		}
		return 0, true
	}
	return count, false
}

// ----------------------------------------------------------------------------
// 换行 / EOF
// ----------------------------------------------------------------------------

func (l *Lexer) consumeNewline() Token {
	tok := l.newToken(NEWLINE, "\\n")
	// \r\n 算一个换行：先吃掉 \r，再吃掉 \n；单独的 \n 或 \r 也各算一个。
	if l.ch == '\r' {
		l.readChar() // 消费 \r（此时 peek 已可能变化）
		if l.ch == '\n' {
			l.readChar()
		}
	} else {
		l.readChar()
	}
	l.atLineStart = true
	return tok
}

func (l *Lexer) handleEOF() Token {
	// 关闭剩余缩进：先弹完所有 DEDENT，让 parser 收尾块结构。
	for len(l.indentStack) > 1 {
		l.indentStack = l.indentStack[:len(l.indentStack)-1]
		l.tokenQueue = append(l.tokenQueue, l.newToken(DEDENT, ""))
	}
	if len(l.tokenQueue) > 0 {
		t := l.tokenQueue[0]
		l.tokenQueue = l.tokenQueue[1:]
		return t
	}
	return l.newToken(EOF, "")
}

// ----------------------------------------------------------------------------
// 词法单元
// ----------------------------------------------------------------------------

func (l *Lexer) lexOneToken() Token {
	switch l.ch {
	// 复合赋值 + / 的两字符变体（// 与 //=）优先于单字符
	case '+', '-', '*', '%':
		if l.peekChar() == '=' {
			tok := l.newToken(augAssignType(l.ch), string(l.ch)+"=")
			l.readChar()
			l.readChar()
			return tok
		}
		tok := l.newToken(singleCharType(l.ch), string(l.ch))
		l.readChar()
		return tok
	case '/':
		if l.peekChar() == '/' {
			// 可能是 //=
			if l.readPosition+1 < len(l.input) && l.input[l.readPosition+1] == '=' {
				l.readChar()
				l.readChar()
				tok := l.newToken(SLASH_SLASH_ASSIGN, "//=")
				l.readChar()
				return tok
			}
			l.readChar()
			tok := l.newToken(SLASH_SLASH, "//")
			l.readChar()
			return tok
		}
		if l.peekChar() == '=' {
			tok := l.newToken(SLASH_ASSIGN, "/=")
			l.readChar()
			l.readChar()
			return tok
		}
		tok := l.newToken(SLASH, "/")
		l.readChar()
		return tok
	case '(', ')', '[', ']', '{', '}', ',', ':', '.':
		tok := l.newToken(singleCharType(l.ch), string(l.ch))
		l.readChar()
		return tok
	case '<', '>', '!', '=':
		return l.consumeTwoCharOp()
	case '"', '\'':
		return l.consumeString()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return l.consumeNumber()
	default:
		if isIdentStart(l.ch) {
			return l.consumeIdentifier()
		}
		tok := l.newToken(ILLEGAL, string(l.ch))
		l.readChar()
		return tok
	}
}

func (l *Lexer) skipInlineWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipToNextLine() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) consumeTwoCharOp() Token {
	cur := l.ch
	switch cur {
	case '=':
		if l.peekChar() == '=' {
			tok := l.newToken(EQ, "==")
			l.readChar()
			l.readChar()
			return tok
		}
		tok := l.newToken(ASSIGN, string(cur))
		l.readChar()
		return tok
	case '!':
		if l.peekChar() == '=' {
			tok := l.newToken(NOT_EQ, "!=")
			l.readChar()
			l.readChar()
			return tok
		}
		tok := l.newToken(ILLEGAL, "!")
		l.readChar()
		return tok
	case '<':
		if l.peekChar() == '=' {
			tok := l.newToken(LT_EQ, "<=")
			l.readChar()
			l.readChar()
			return tok
		}
		tok := l.newToken(LT, string(cur))
		l.readChar()
		return tok
	case '>':
		if l.peekChar() == '=' {
			tok := l.newToken(GT_EQ, ">=")
			l.readChar()
			l.readChar()
			return tok
		}
		tok := l.newToken(GT, string(cur))
		l.readChar()
		return tok
	}
	tok := l.newToken(ILLEGAL, string(cur))
	l.readChar()
	return tok
}

func (l *Lexer) consumeNumber() Token {
	start := l.position
	isFloat := false
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	lit := l.input[start:l.position]
	if isFloat {
		return l.newToken(FLOAT, lit)
	}
	return l.newToken(INT, lit)
}

func (l *Lexer) consumeString() Token {
	quote := l.ch
	startLine, startCol := l.line, l.col
	start := l.position + 1
	l.readChar()
	for l.ch != quote && l.ch != 0 && l.ch != '\n' {
		if l.ch == '\\' {
			l.readChar()
		}
		l.readChar()
	}
	if l.ch != quote {
		return Token{Type: ILLEGAL, Literal: "unterminated string", Line: startLine, Column: startCol}
	}
	raw := l.input[start:l.position]
	l.readChar()
	return Token{Type: STRING, Literal: unescapeString(raw), Line: startLine, Column: startCol}
}

func unescapeString(s string) string {
	r := strings.NewReplacer(
		`\n`, "\n",
		`\t`, "\t",
		`\\`, "\\",
		`\"`, "\"",
		`\'`, "'",
	)
	return r.Replace(s)
}

func (l *Lexer) consumeIdentifier() Token {
	start := l.position
	for isIdentPart(l.ch) {
		l.readChar()
	}
	lit := l.input[start:l.position]
	tokType := LookupIdent(lit)
	return l.newToken(tokType, lit)
}

// ----------------------------------------------------------------------------
// 工具
// ----------------------------------------------------------------------------

func (l *Lexer) newToken(typ TokenType, lit string) Token {
	return Token{Type: typ, Literal: lit, Line: l.line, Column: l.col}
}

func singleCharType(ch byte) TokenType {
	switch ch {
	case '+':
		return PLUS
	case '-':
		return MINUS
	case '*':
		return ASTERISK
	case '/':
		return SLASH
	case '%':
		return PERCENT
	case '(':
		return LPAREN
	case ')':
		return RPAREN
	case '[':
		return LBRACKET
	case ']':
		return RBRACKET
	case '{':
		return LBRACE
	case '}':
		return RBRACE
	case '.':
		return DOT
	case ',':
		return COMMA
	case ':':
		return COLON
	}
	return ILLEGAL
}

// augAssignType 把复合赋值的首字符映射到对应 token 类型。
func augAssignType(ch byte) TokenType {
	switch ch {
	case '+':
		return PLUS_ASSIGN
	case '-':
		return MINUS_ASSIGN
	case '*':
		return ASTERISK_ASSIGN
	case '%':
		return PERCENT_ASSIGN
	}
	return ILLEGAL
}

func isDigit(ch byte) bool      { return ch >= '0' && ch <= '9' }
func isIdentStart(ch byte) bool  { return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch >= 0x80 }
func isIdentPart(ch byte) bool   { return isIdentStart(ch) || isDigit(ch) }

// 仅供错误信息使用：让 token 实现 fmt.Formatter。
func (t Token) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, "%s(%q) at %d:%d", t.Type, t.Literal, t.Line, t.Column)
}