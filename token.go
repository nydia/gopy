package main

// TokenType 标识 token 的种类。Python 子集需要的关键字、运算符、字面量都在这里。
type TokenType string

const (
	// 特殊
	ILLEGAL TokenType = "ILLEGAL" // 非法字符
	EOF     TokenType = "EOF"     // 文件结束
	NEWLINE TokenType = "NEWLINE" // 换行（语句分隔）
	INDENT  TokenType = "INDENT"  // 缩进增加
	DEDENT  TokenType = "DEDENT"  // 缩进减少

	// 字面量
	IDENT  TokenType = "IDENT"  // 标识符（变量名、函数名）
	INT    TokenType = "INT"    // 整数
	FLOAT  TokenType = "FLOAT"  // 浮点数
	STRING TokenType = "STRING" // 字符串

	// 运算符
	ASSIGN   TokenType = "="
	PLUS     TokenType = "+"
	MINUS    TokenType = "-"
	ASTERISK TokenType = "*"
	SLASH    TokenType = "/"
	SLASH_SLASH TokenType = "//" // 整除
	PERCENT  TokenType = "%"

	// 复合赋值
	PLUS_ASSIGN        TokenType = "+="
	MINUS_ASSIGN       TokenType = "-="
	ASTERISK_ASSIGN    TokenType = "*="
	SLASH_ASSIGN       TokenType = "/="
	SLASH_SLASH_ASSIGN TokenType = "//="
	PERCENT_ASSIGN     TokenType = "%="

	EQ     TokenType = "=="
	NOT_EQ TokenType = "!="
	LT     TokenType = "<"
	GT     TokenType = ">"
	LT_EQ  TokenType = "<="
	GT_EQ  TokenType = ">="

	// 分隔符
	COMMA     TokenType = ","
	COLON     TokenType = ":"
	LPAREN    TokenType = "("
	RPAREN    TokenType = ")"
	LBRACKET  TokenType = "["
	RBRACKET  TokenType = "]"
	LBRACE    TokenType = "{"
	RBRACE    TokenType = "}"
	DOT       TokenType = "."

	// 关键字
	DEF    TokenType = "DEF"
	RETURN TokenType = "RETURN"
	IF     TokenType = "IF"
	ELIF   TokenType = "ELIF"
	ELSE   TokenType = "ELSE"
	WHILE  TokenType = "WHILE"
	FOR    TokenType = "FOR"
	IN     TokenType = "IN"

	TRUE   TokenType = "TRUE"
	FALSE  TokenType = "FALSE"
	NONE   TokenType = "NONE"
	AND    TokenType = "AND"
	OR     TokenType = "OR"
	NOT    TokenType = "NOT"
	PASS   TokenType = "PASS"
	BREAK  TokenType = "BREAK"
	CONTINUE TokenType = "CONTINUE"
	TRY    TokenType = "TRY"
	EXCEPT TokenType = "EXCEPT"
	RAISE  TokenType = "RAISE"
	AS     TokenType = "AS"
)

// keywords 是关键字字面量到 TokenType 的映射。词法分析时遇到 IDENT
// 时查表决定是关键字还是普通标识符。
var keywords = map[string]TokenType{
	"def":     DEF,
	"return":  RETURN,
	"if":      IF,
	"elif":    ELIF,
	"else":    ELSE,
	"while":   WHILE,
	"for":     FOR,
	"in":      IN,
	"True":    TRUE,
	"False":   FALSE,
	"None":    NONE,
	"and":     AND,
	"or":      OR,
	"not":     NOT,
	"pass":    PASS,
	"break":   BREAK,
	"continue": CONTINUE,
	"try":     TRY,
	"except":  EXCEPT,
	"raise":   RAISE,
	"as":      AS,
}

// LookupIdent 判断一个标识符是不是关键字。
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// Token 是词法分析器输出的最小单元。
type Token struct {
	Type    TokenType
	Literal string
	Line    int    // 行号（用于错误信息）
	Column  int    // 列号
	Indent  int    // 当前缩进级别（用于 INDENT/DEDENT）
}