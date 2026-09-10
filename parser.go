package main

import (
	"fmt"
	"os"
	"strconv"
)

// ----------------------------------------------------------------------------
// 优先级表
// ----------------------------------------------------------------------------

const (
	_ int = iota
	LOWEST
	TERNARY     // a if cond else b
	LOGICAL_OR   // or
	LOGICAL_AND  // and
	LOGICAL_NOT  // not
	EQUALS       // == != in not in
	LESSGREATER  // < > <= >=
	SUM          // + -
	PRODUCT      // * / %
	PREFIX       // -x  not x
	CALL         // f(x)
	INDEX        // arr[i]
	METHOD       // obj.attr
)

var precedences = map[TokenType]int{
	EQ:       EQUALS,
	NOT_EQ:   EQUALS,
	IN:       EQUALS,
	LT:       LESSGREATER,
	GT:       LESSGREATER,
	LT_EQ:    LESSGREATER,
	GT_EQ:    LESSGREATER,
	PLUS:     SUM,
	MINUS:    SUM,
	ASTERISK: PRODUCT,
	SLASH:    PRODUCT,
	SLASH_SLASH: PRODUCT,
	PERCENT:  PRODUCT,
	AND:      LOGICAL_AND,
	OR:       LOGICAL_OR,
	IF:       TERNARY,
	LPAREN:   CALL,
	LBRACKET: INDEX,
	DOT:      METHOD,
}

// ----------------------------------------------------------------------------
// Parser
// ----------------------------------------------------------------------------

type prefixParseFn func() Expression
type infixParseFn func(Expression) Expression

// Parser 用 Pratt 解析法把 token 流转换成 Program。
type Parser struct {
	l      *Lexer
	errors []string

	curToken  Token
	peekToken Token

	// noIn 为 true 时，顶层的 `in` 不当作成员运算符消耗
	//（for 语句头里的 iterable 表达式；括号分组会临时清掉该标志）。
	noIn bool

	prefixParseFns map[TokenType]prefixParseFn
	infixParseFns  map[TokenType]infixParseFn
}

// NewParser 由 lexer 构造一个 Parser。
func NewParser(l *Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}
	p.prefixParseFns = make(map[TokenType]prefixParseFn)
	p.infixParseFns = make(map[TokenType]infixParseFn)
	p.registerFns()

	// 读两个 token 填满窗口
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) registerFns() {
	// 前缀
	p.prefixParseFns[IDENT] = p.parseIdentifier
	p.prefixParseFns[INT] = p.parseIntegerLiteral
	p.prefixParseFns[FLOAT] = p.parseFloatLiteral
	p.prefixParseFns[STRING] = p.parseStringLiteral
	p.prefixParseFns[TRUE] = p.parseBooleanLiteral
	p.prefixParseFns[FALSE] = p.parseBooleanLiteral
	p.prefixParseFns[NONE] = p.parseNoneLiteral
	p.prefixParseFns[MINUS] = p.parsePrefixExpression
	p.prefixParseFns[NOT] = p.parsePrefixExpression
	p.prefixParseFns[LPAREN] = p.parseGroupedExpression
	p.prefixParseFns[LBRACKET] = p.parseArrayLiteral
	p.prefixParseFns[LBRACE] = p.parseDictLiteral

	// 中缀
	p.infixParseFns[PLUS] = p.parseInfixExpression
	p.infixParseFns[MINUS] = p.parseInfixExpression
	p.infixParseFns[ASTERISK] = p.parseInfixExpression
	p.infixParseFns[SLASH] = p.parseInfixExpression
	p.infixParseFns[SLASH_SLASH] = p.parseInfixExpression
	p.infixParseFns[PERCENT] = p.parseInfixExpression
	p.infixParseFns[EQ] = p.parseInfixExpression
	p.infixParseFns[NOT_EQ] = p.parseInfixExpression
	p.infixParseFns[LT] = p.parseInfixExpression
	p.infixParseFns[GT] = p.parseInfixExpression
	p.infixParseFns[LT_EQ] = p.parseInfixExpression
	p.infixParseFns[GT_EQ] = p.parseInfixExpression
	p.infixParseFns[AND] = p.parseInfixExpression
	p.infixParseFns[OR] = p.parseInfixExpression
	p.infixParseFns[IN] = p.parseInfixExpression
	p.infixParseFns[LPAREN] = p.parseCallExpression
	p.infixParseFns[LBRACKET] = p.parseIndexExpression
	p.infixParseFns[DOT] = p.parseAttributeExpression
	p.infixParseFns[IF] = p.parseIfExpressionInfix
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// ----------------------------------------------------------------------------
// 谓词 / 错误
// ----------------------------------------------------------------------------

func (p *Parser) curTokenIs(t TokenType) bool  { return p.curToken.Type == t }
func (p *Parser) peekTokenIs(t TokenType) bool { return p.peekToken.Type == t }

func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) Errors() []string { return p.errors }

func (p *Parser) peekError(t TokenType) {
	p.errors = append(p.errors, fmt.Sprintf(
		"expected next token to be %s, got %s (%q) at %d:%d",
		t, p.peekToken.Type, p.peekToken.Literal, p.peekToken.Line, p.peekToken.Column))
}

func (p *Parser) noPrefixParseFnError(t TokenType) {
	if t == NEWLINE || t == EOF || t == DEDENT {
		return
	}
	p.errors = append(p.errors, fmt.Sprintf(
		"no prefix parse function for %s (%q) at %d:%d",
		t, p.curToken.Literal, p.curToken.Line, p.curToken.Column))
}

// ----------------------------------------------------------------------------
// Program / Statements
// ----------------------------------------------------------------------------

// ParseProgram 是入口，返回根 Program。
// 约定：parseStatement 返回时 cur 在语句最后一个 token 之后的 token 上
// （即已 nextToken 一次推到 NEWLINE/EOF/下一条语句起点）。
func (p *Parser) ParseProgram() *Program {
	program := &Program{Statements: []Statement{}}

	for !p.curTokenIs(EOF) {
		if p.curToken.Type == INDENT || p.curToken.Type == DEDENT || p.curToken.Type == NEWLINE {
			p.nextToken()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		// parseStatement 已经把 cur 推到语句之后；这里跳过空白 token。
		for !p.curTokenIs(EOF) &&
			(p.curToken.Type == INDENT || p.curToken.Type == DEDENT || p.curToken.Type == NEWLINE) {
			p.nextToken()
		}
	}

	return program
}

// augAssignOps 把复合赋值 token 映射到归一化运算符（evaluator 直接复用中缀求值）。
var augAssignOps = map[TokenType]string{
	PLUS_ASSIGN:        "+",
	MINUS_ASSIGN:       "-",
	ASTERISK_ASSIGN:    "*",
	SLASH_ASSIGN:       "/",
	SLASH_SLASH_ASSIGN: "//",
	PERCENT_ASSIGN:     "%",
}

// parseStatement 决定具体语句类型并分发。
func (p *Parser) parseStatement() Statement {
	switch p.curToken.Type {
	case IDENT:
		if p.peekTokenIs(ASSIGN) {
			return p.parseLetStatement()
		}
		// x += 1 之类的复合赋值
		if _, isAug := augAssignOps[p.peekToken.Type]; isAug {
			return p.parseAugAssignStatement()
		}
		// a, b = expr 序列解包赋值
		if p.peekTokenIs(COMMA) {
			return p.parseUnpackAssignStatement()
		}
		// arr[i] = x 走这里（parseExpressionStatement 里识别）
		return p.parseExpressionStatement()
	case IF:
		return p.parseIfStatement()
	case RETURN:
		return p.parseReturnStatement()
	case WHILE:
		return p.parseWhileStatement()
	case FOR:
		return p.parseForStatement()
	case DEF:
		return p.parseFunctionStatement()
	case BREAK:
		tok := p.curToken
		p.nextToken()
		return &BreakStatement{Token: tok}
	case CONTINUE:
		tok := p.curToken
		p.nextToken()
		return &ContinueStatement{Token: tok}
	case PASS:
		tok := p.curToken
		p.nextToken()
		return &PassStatement{Token: tok}
	case TRY:
		return p.parseTryStatement()
	case RAISE:
		return p.parseRaiseStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// parseBlockStatement 进入时 curToken 必须是 INDENT。
// 结束时 curToken = DEDENT（调用方决定要不要 nextToken）。
func (p *Parser) parseBlockStatement() *BlockStatement {
	block := &BlockStatement{Token: p.curToken}
	block.Statements = []Statement{}
	p.nextToken() // 跳过 INDENT

	for !p.curTokenIs(DEDENT) && !p.curTokenIs(EOF) {
		if p.curToken.Type == NEWLINE {
			p.nextToken()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}
	// 停在 DEDENT 上，由调用方 nextToken
	return block
}

// parseLetStatement：name = expr
func (p *Parser) parseLetStatement() *LetStatement {
	stmt := &LetStatement{Token: p.curToken}
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(ASSIGN) {
		return nil
	}
	p.nextToken() // 跨过 '='
	stmt.Value = p.parseExpression(LOWEST)
	if stmt.Value == nil {
		return nil
	}
	p.nextToken() // 跨过 RHS 最后 token
	return stmt
}

// parseReturnStatement：return [expr]
func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{Token: p.curToken}
	p.nextToken()
	if p.curTokenIs(NEWLINE) || p.curTokenIs(EOF) || p.curTokenIs(DEDENT) {
		return stmt
	}
	stmt.ReturnValue = p.parseExpression(LOWEST)
	if stmt.ReturnValue != nil {
		p.nextToken() // 跨过返回值最后 token
	}
	return stmt
}

// parseIfStatement：if cond: block (elif cond: block)* (else: block)?
//
// 约定：parseBlockStatement 返回时 cur=DEDENT（consequence / elif / alternative 各自的
// 结束 DEDENT）。所以这里统一通过 peekToken 判断下一段是否是 ELIF / ELSE，需要时
// nextToken 推进一次穿过中间的 DEDENT。函数返回时 cur 停在最后一个 DEDENT 上，
// 由调用方（外层 block 或 ParseProgram）消费。
func (p *Parser) parseIfStatement() *IfStatement {
	stmt := &IfStatement{Token: p.curToken}
	p.nextToken() // 跳过 if
	stmt.Condition = p.parseExpression(LOWEST)
	if stmt.Condition == nil {
		return nil
	}
	if !p.expectPeek(COLON) {
		return nil
	}
	if !p.expectPeek(NEWLINE) {
		return nil
	}
	if !p.expectPeek(INDENT) {
		return nil
	}
	stmt.Consequence = p.parseBlockStatement()
	// cur = DEDENT（consequence 结束）

	for p.peekTokenIs(ELIF) {
		p.nextToken() // 穿过 DEDENT，cur = ELIF
		elifTok := p.curToken
		p.nextToken() // cur = 条件起点
		cond := p.parseExpression(LOWEST)
		if !p.expectPeek(COLON) {
			return nil
		}
		if !p.expectPeek(NEWLINE) {
			return nil
		}
		if !p.expectPeek(INDENT) {
			return nil
		}
		body := p.parseBlockStatement()
		// cur = DEDENT（elif body 结束）
		stmt.Elifs = append(stmt.Elifs, &ElifClause{
			Token: elifTok, Condition: cond, Consequence: body,
		})
	}

	if p.peekTokenIs(ELSE) {
		p.nextToken() // 穿过 DEDENT，cur = ELSE
		// cur = ELSE，peek 应该是 ':'
		if !p.expectPeek(COLON) {
			return nil
		}
		if !p.expectPeek(NEWLINE) {
			return nil
		}
		if !p.expectPeek(INDENT) {
			return nil
		}
		stmt.Alternative = p.parseBlockStatement()
		// cur = DEDENT（alternative 结束）
	}
	// cur = DEDENT，由外层消费
	return stmt
}

// parseWhileStatement：while cond: block
func (p *Parser) parseWhileStatement() *WhileStatement {
	stmt := &WhileStatement{Token: p.curToken}
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)
	if !p.expectPeek(COLON) {
		return nil
	}
	if !p.expectPeek(NEWLINE) {
		return nil
	}
	if !p.expectPeek(INDENT) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	// cur = DEDENT
	return stmt
}

// parseForStatement：for x in iter: block 或 for k, v in iter: block
//
// 约定与 parseWhileStatement 一致：parseBlockStatement 返回时 cur=DEDENT，
// 由 ParseProgram 的空白 token 消费循环处理。
func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{Token: p.curToken}
	p.nextToken() // 跳过 for
	if !p.curTokenIs(IDENT) {
		p.errors = append(p.errors, fmt.Sprintf(
			"expected identifier after 'for' at %d:%d", p.curToken.Line, p.curToken.Column))
		return nil
	}
	stmt.Targets = []*Identifier{{Token: p.curToken, Value: p.curToken.Literal}}
	p.nextToken() // cur = ',' 或 IN
	for p.curTokenIs(COMMA) {
		p.nextToken() // cur = 下一个目标
		if !p.curTokenIs(IDENT) {
			p.errors = append(p.errors, fmt.Sprintf(
				"expected identifier in for target list at %d:%d",
				p.curToken.Line, p.curToken.Column))
			return nil
		}
		stmt.Targets = append(stmt.Targets, &Identifier{Token: p.curToken, Value: p.curToken.Literal})
		p.nextToken() // cur = ',' 或 IN
	}
	if !p.curTokenIs(IN) {
		p.errors = append(p.errors, fmt.Sprintf(
			"expected 'in' in for statement at %d:%d", p.curToken.Line, p.curToken.Column))
		return nil
	}
	// 迭代表达式：置 noIn，防止顶层 `in` 被当作成员运算符吞掉
	p.noIn = true
	p.nextToken() // cur = iterable 起点
	stmt.Iter = p.parseExpression(LOWEST)
	p.noIn = false
	if !p.expectPeek(COLON) {
		return nil
	}
	if !p.expectPeek(NEWLINE) {
		return nil
	}
	if !p.expectPeek(INDENT) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	// cur = DEDENT（for body 结束）
	return stmt
}

// parseUnpackAssignStatement：a, b = expr；RHS 为逗号分隔时包成隐式元组。
// 进入时 cur=首个目标 IDENT，peek=','。
func (p *Parser) parseUnpackAssignStatement() Statement {
	stmt := &UnpackAssignStatement{Token: p.curToken}
	stmt.Targets = []*Identifier{{Token: p.curToken, Value: p.curToken.Literal}}
	p.nextToken() // cur = ','
	for p.curTokenIs(COMMA) {
		p.nextToken() // cur = 下一个目标
		if !p.curTokenIs(IDENT) {
			p.errors = append(p.errors, fmt.Sprintf(
				"expected identifier in assignment target list at %d:%d",
				p.curToken.Line, p.curToken.Column))
			return nil
		}
		stmt.Targets = append(stmt.Targets, &Identifier{Token: p.curToken, Value: p.curToken.Literal})
		p.nextToken() // cur = ',' 或 '='
	}
	if !p.curTokenIs(ASSIGN) {
		p.errors = append(p.errors, fmt.Sprintf(
			"expected '=' in assignment at %d:%d", p.curToken.Line, p.curToken.Column))
		return nil
	}
	p.nextToken() // cur = RHS 起点
	first := p.parseExpression(LOWEST)
	if first == nil {
		return nil
	}
	if p.peekTokenIs(COMMA) {
		// 隐式元组 RHS：a, b = 1, 2
		elems := []Expression{first}
		for p.peekTokenIs(COMMA) {
			p.nextToken() // cur = ','
			p.nextToken() // cur = 下一个表达式
			e := p.parseExpression(LOWEST)
			if e == nil {
				return nil
			}
			elems = append(elems, e)
		}
		stmt.Value = &TupleLiteral{Token: stmt.Token, Elements: elems}
	} else {
		stmt.Value = first
	}
	p.nextToken() // 跨过 RHS 最后 token
	return stmt
}

// parseFunctionStatement：def name(params): block
func (p *Parser) parseFunctionStatement() *FunctionStatement {
	stmt := &FunctionStatement{Token: p.curToken}
	p.nextToken() // 跳过 def
	if !p.curTokenIs(IDENT) {
		p.errors = append(p.errors, "expected function name after 'def'")
		return nil
	}
	stmt.Name = &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(LPAREN) {
		return nil
	}
	stmt.Parameters = p.parseFunctionParameters()
	if stmt.Parameters == nil {
		return nil
	}
	if !p.expectPeek(COLON) {
		return nil
	}
	if !p.expectPeek(NEWLINE) {
		return nil
	}
	if !p.expectPeek(INDENT) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	// cur = DEDENT
	return stmt
}

// parseFunctionParameters 解析 (a, b=2, c)。
// 校验:非默认参数不能跟在默认参数之后(CPython 规则)。
func (p *Parser) parseFunctionParameters() []*Parameter {
	params := []*Parameter{}
	p.nextToken() // 跨过 '('
	if p.curTokenIs(RPAREN) {
		return params
	}
	seenDefault := false
	for {
		if !p.curTokenIs(IDENT) {
			p.errors = append(p.errors, fmt.Sprintf(
				"expected parameter name at %d:%d", p.curToken.Line, p.curToken.Column))
			return nil
		}
		param := &Parameter{Name: &Identifier{Token: p.curToken, Value: p.curToken.Literal}}
		if p.peekTokenIs(ASSIGN) {
			p.nextToken() // cur = '='
			p.nextToken() // cur = 默认值表达式起点
			d := p.parseExpression(LOWEST)
			if d == nil {
				return nil
			}
			param.Default = d
			seenDefault = true
		} else if seenDefault {
			p.errors = append(p.errors, fmt.Sprintf(
				"non-default argument follows default argument at %d:%d",
				p.curToken.Line, p.curToken.Column))
			return nil
		}
		params = append(params, param)
		p.nextToken() // cur = ',' 或 ')'
		if !p.curTokenIs(COMMA) {
			break
		}
		p.nextToken() // cur = 下一个参数名
	}
	if !p.curTokenIs(RPAREN) {
		p.errors = append(p.errors, fmt.Sprintf(
			"expected ')' in parameter list at %d:%d", p.curToken.Line, p.curToken.Column))
		return nil
	}
	return params
}

// parseAugAssignStatement：x op= expr（进入时 cur=IDENT，peek=复合赋值 token）
func (p *Parser) parseAugAssignStatement() *AugAssignStatement {
	stmt := &AugAssignStatement{Token: p.peekToken}
	stmt.Target = &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	op, ok := augAssignOps[p.peekToken.Type]
	if !ok {
		p.peekError(ASSIGN)
		return nil
	}
	stmt.Operator = op
	p.nextToken() // cur = op token
	p.nextToken() // cur = RHS 起点
	stmt.Value = p.parseExpression(LOWEST)
	if stmt.Value == nil {
		return nil
	}
	p.nextToken() // 跨过 RHS 最后 token
	return stmt
}

// parseTryStatement：try: block (except [Kind [as name]]: block)+
// 进入时 cur='try'，返回时 cur 停在最后一个 handler 的 DEDENT 上。
func (p *Parser) parseTryStatement() *TryStatement {
	stmt := &TryStatement{Token: p.curToken}
	if !p.expectPeek(COLON) {
		return nil
	}
	if !p.expectPeek(NEWLINE) {
		return nil
	}
	if !p.expectPeek(INDENT) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()
	// cur = DEDENT

	for p.peekTokenIs(EXCEPT) {
		p.nextToken() // 穿过 DEDENT，cur = except
		clause := &ExceptClause{Token: p.curToken}
		p.nextToken() // cur = ':' (裸 except) 或 异常类型名
		if p.curTokenIs(IDENT) {
			clause.Kind = p.curToken.Literal
			if p.peekTokenIs(AS) {
				p.nextToken() // cur = as
				if !p.expectPeek(IDENT) {
					return nil
				}
				clause.BindName = p.curToken.Literal
			}
			if !p.expectPeek(COLON) {
				return nil
			}
		} else if !p.curTokenIs(COLON) {
			p.errors = append(p.errors, fmt.Sprintf(
				"expected exception type after 'except' at %d:%d",
				p.curToken.Line, p.curToken.Column))
			return nil
		}
		if !p.expectPeek(NEWLINE) {
			return nil
		}
		if !p.expectPeek(INDENT) {
			return nil
		}
		clause.Body = p.parseBlockStatement()
		// cur = DEDENT
		stmt.Handlers = append(stmt.Handlers, clause)
	}
	return stmt
}

// parseRaiseStatement：raise expr（进入时 cur='raise'）
func (p *Parser) parseRaiseStatement() *RaiseStatement {
	stmt := &RaiseStatement{Token: p.curToken}
	p.nextToken()
	if p.curTokenIs(NEWLINE) || p.curTokenIs(EOF) || p.curTokenIs(DEDENT) {
		// bare raise：当前没有活动异常可重抛，evaluator 会报错
		return stmt
	}
	stmt.Value = p.parseExpression(LOWEST)
	if stmt.Value != nil {
		p.nextToken() // 跨过表达式最后 token
	}
	return stmt
}

// parseExpressionStatement 把当前 token 视作表达式起点。
// 顺便把 arr[i] = x 转成 IndexAssignExpression。
func (p *Parser) parseExpressionStatement() *ExpressionStatement {
	tok := p.curToken
	stmt := &ExpressionStatement{Token: tok}
	stmt.Expression = p.parseExpression(LOWEST)
	if stmt.Expression == nil {
		p.nextToken() // 推进避免 ParseProgram 死循环
		return nil
	}
	if idx, ok := stmt.Expression.(*IndexExpression); ok {
		if p.peekTokenIs(ASSIGN) {
			p.nextToken() // cur = =
			p.nextToken() // cur = RHS 起点
			rhs := p.parseExpression(LOWEST)
			if rhs == nil {
				return nil
			}
			stmt.Expression = &IndexAssignExpression{
				Token:  tok,
				Target: idx.Left,
				Index:  idx.Index,
				Value:  rhs,
			}
		} else if op, isAug := augAssignOps[p.peekToken.Type]; isAug {
			// arr[i] op= x → arr[i] = arr[i] op x（左目标作为值参与运算）
			opTok := p.peekToken
			p.nextToken() // cur = op
			p.nextToken() // cur = RHS 起点
			rhs := p.parseExpression(LOWEST)
			if rhs == nil {
				return nil
			}
			stmt.Expression = &IndexAssignExpression{
				Token:  tok,
				Target: idx.Left,
				Index:  idx.Index,
				Value: &InfixExpression{
					Token: opTok, Left: idx, Operator: op, Right: rhs,
				},
			}
		}
	}
	p.nextToken() // 跨过表达式最后 token
	return stmt
}

// ----------------------------------------------------------------------------
// Pratt 表达式解析
// ----------------------------------------------------------------------------

func (p *Parser) parseExpression(precedence int) Expression {
	if os.Getenv("GOPY_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[parseExpression] enter cur=%v %q, precedence=%d\n",
			p.curToken.Type, p.curToken.Literal, precedence)
	}
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()
	if leftExp == nil {
		return nil
	}

	for {
		// for 语句头：顶层的 in 属于 for 语法，不当作成员运算符
		if p.noIn && p.peekTokenIs(IN) {
			break
		}

		// 'not in' 是双词运算符，在任何优先级窗口都要先于普通 infix 检查。
		// 仅当上下文优先级低于 EQUALS 时绑定（与 in 同级、左结合）。
		if p.peekTokenIs(NOT) && precedence < EQUALS {
			notTok := p.peekToken
			p.nextToken() // cur = not
			if !p.peekTokenIs(IN) {
				p.errors = append(p.errors, fmt.Sprintf(
					"expected 'in' after 'not' in membership test at %d:%d",
					p.curToken.Line, p.curToken.Column))
				return nil
			}
			p.nextToken() // cur = in
			p.nextToken() // cur = 右操作数起点
			right := p.parseExpression(EQUALS)
			if right == nil {
				return nil
			}
			leftExp = &InfixExpression{
				Token: notTok, Left: leftExp, Operator: "not in", Right: right,
			}
			continue
		}

		if p.peekTokenIs(NEWLINE) || p.peekTokenIs(EOF) || p.peekTokenIs(DEDENT) ||
			precedence >= p.peekPrecedence() {
			break
		}
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		p.nextToken()
		leftExp = infix(leftExp)
		if leftExp == nil {
			return nil
		}
	}
	return leftExp
}

func (p *Parser) peekPrecedence() int {
	if pr, ok := precedences[p.peekToken.Type]; ok {
		return pr
	}
	return LOWEST
}

// ----------------------------------------------------------------------------
// 前缀解析函数
// ----------------------------------------------------------------------------

func (p *Parser) parseIdentifier() Expression {
	return &Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() Expression {
	lit := &IntegerLiteral{Token: p.curToken}
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf(
			"could not parse %q as integer at %d:%d",
			p.curToken.Literal, p.curToken.Line, p.curToken.Column))
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() Expression {
	lit := &FloatLiteral{Token: p.curToken}
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf(
			"could not parse %q as float at %d:%d",
			p.curToken.Literal, p.curToken.Line, p.curToken.Column))
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() Expression {
	return &StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() Expression {
	return &BooleanLiteral{Token: p.curToken, Value: p.curToken.Literal == "True"}
}

func (p *Parser) parseNoneLiteral() Expression {
	return &NoneLiteral{Token: p.curToken}
}

func (p *Parser) parsePrefixExpression() Expression {
	expr := &PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}
	p.nextToken()
	expr.Right = p.parseExpression(PREFIX)
	return expr
}

// parseGroupedExpression 处理 (a) 分组与 (a, b) / (a,) / () 元组字面量。
// 进入时 curToken 是 '('。
func (p *Parser) parseGroupedExpression() Expression {
	openTok := p.curToken

	// 括号内恢复 `in` 的成员运算符身份（如 for x in (a in b):）
	savedNoIn := p.noIn
	p.noIn = false
	defer func() { p.noIn = savedNoIn }()

	p.nextToken() // 跨过 (
	if p.curTokenIs(RPAREN) {
		return &TupleLiteral{Token: openTok, Elements: []Expression{}}
	}
	first := p.parseExpression(LOWEST)
	if first == nil {
		return nil
	}
	if !p.peekTokenIs(COMMA) {
		if !p.expectPeek(RPAREN) {
			return nil
		}
		return first // 纯分组表达式
	}
	// 元组：(expr, ) 或 (e1, e2, ...)
	elems := []Expression{first}
	for p.peekTokenIs(COMMA) {
		p.nextToken() // cur = ','
		if p.peekTokenIs(RPAREN) {
			break // 尾逗号 (1,)
		}
		p.nextToken() // cur = 下一个表达式起点
		e := p.parseExpression(LOWEST)
		if e == nil {
			return nil
		}
		elems = append(elems, e)
	}
	if !p.expectPeek(RPAREN) {
		return nil
	}
	return &TupleLiteral{Token: openTok, Elements: elems}
}

// parseArrayLiteral：[a, b, c]
func (p *Parser) parseArrayLiteral() Expression {
	arr := &ArrayLiteral{Token: p.curToken}
	arr.Elements = p.parseExpressionList(RBRACKET)
	if arr.Elements == nil {
		return nil
	}
	return arr
}

// parseDictLiteral：{k1: v1, k2: v2} 或空 {}。
// 进入时 curToken 是 '{'，返回时 curToken 是 '}'。
func (p *Parser) parseDictLiteral() Expression {
	dict := &DictLiteral{Token: p.curToken}
	dict.Keys = []Expression{}
	dict.Values = []Expression{}
	p.nextToken() // 跨过 '{'
	if p.curTokenIs(RBRACE) {
		return dict
	}
	for {
		k := p.parseExpression(LOWEST)
		if k == nil {
			return nil
		}
		if !p.expectPeek(COLON) {
			return nil
		}
		p.nextToken() // 跨过 ':'
		v := p.parseExpression(LOWEST)
		if v == nil {
			return nil
		}
		dict.Keys = append(dict.Keys, k)
		dict.Values = append(dict.Values, v)
		if p.peekTokenIs(COMMA) {
			p.nextToken() // cur = ','
			p.nextToken() // cur = 下一个键的起点
			continue
		}
		break
	}
	if !p.expectPeek(RBRACE) {
		return nil
	}
	return dict
}

// parseExpressionList 解析由逗号分隔、用 end 结束的表达式列表（用于函数参数、数组）。
// 进入时 curToken 是 '[' 或 '('。
func (p *Parser) parseExpressionList(end TokenType) []Expression {
	list := []Expression{}
	p.nextToken() // 跨过 [ 或 (
	if p.curTokenIs(end) {
		return list
	}
	list = append(list, p.parseExpression(LOWEST))
	for p.peekTokenIs(COMMA) {
		p.nextToken() // cur = ,
		p.nextToken() // cur = 下一个表达式起点
		list = append(list, p.parseExpression(LOWEST))
	}
	if !p.expectPeek(end) {
		return nil
	}
	return list
}

// ----------------------------------------------------------------------------
// 中缀解析函数
// ----------------------------------------------------------------------------

func (p *Parser) parseInfixExpression(left Expression) Expression {
	expr := &InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)
	return expr
}

func (p *Parser) curPrecedence() int {
	if pr, ok := precedences[p.curToken.Type]; ok {
		return pr
	}
	return LOWEST
}

// parseCallExpression 解析 f(a, b) 与 f(a, key=val) 混合实参。
// 校验:位置参数不能出现在关键字参数之后(CPython 规则)。
func (p *Parser) parseCallExpression(function Expression) Expression {
	exp := &CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = []Expression{}
	exp.ArgNames = []string{}
	p.nextToken() // 跨过 '('
	if p.curTokenIs(RPAREN) {
		return exp
	}
	seenKeyword := false
	for {
		if p.curTokenIs(IDENT) && p.peekTokenIs(ASSIGN) {
			seenKeyword = true
			exp.ArgNames = append(exp.ArgNames, p.curToken.Literal)
			p.nextToken() // cur = '='
			p.nextToken() // cur = 值表达式起点
		} else {
			if seenKeyword {
				p.errors = append(p.errors, fmt.Sprintf(
					"positional argument follows keyword argument at %d:%d",
					p.curToken.Line, p.curToken.Column))
				return nil
			}
			exp.ArgNames = append(exp.ArgNames, "")
		}
		e := p.parseExpression(LOWEST)
		if e == nil {
			return nil
		}
		exp.Arguments = append(exp.Arguments, e)
		if p.peekTokenIs(COMMA) {
			p.nextToken() // cur = ','
			p.nextToken() // cur = 下一个参数起点
			continue
		}
		break
	}
	if !p.expectPeek(RPAREN) {
		return nil
	}
	return exp
}

// parseIndexExpression 解析 arr[i] 与切片 arr[a:b:c]。
// 进入时 curToken 是 '['，三段 start/stop/step 均可省略。
// 约定:parseExpression 返回时 cur 停在表达式最后一个 token 上，
// 所以下一段分隔符 ':' 需要经 peek 判断后再 nextToken 推进。
func (p *Parser) parseIndexExpression(left Expression) Expression {
	openTok := p.curToken
	p.nextToken() // cur = 内容起点 / ':' / ']'
	var start, stop, step Expression
	isSlice := false

	if !p.curTokenIs(COLON) && !p.curTokenIs(RBRACKET) {
		e := p.parseExpression(LOWEST)
		if e == nil {
			return nil
		}
		start = e
		if p.peekTokenIs(COLON) {
			p.nextToken() // cur = ':'
		}
	}
	if p.curTokenIs(COLON) {
		isSlice = true
		p.nextToken() // cur = stop 起点 / ':' / ']'
		if !p.curTokenIs(COLON) && !p.curTokenIs(RBRACKET) {
			e := p.parseExpression(LOWEST)
			if e == nil {
				return nil
			}
			stop = e
			if p.peekTokenIs(COLON) {
				p.nextToken() // cur = ':'
			}
		}
		if p.curTokenIs(COLON) {
			p.nextToken() // cur = step 起点 / ']'
			if !p.curTokenIs(RBRACKET) {
				e := p.parseExpression(LOWEST)
				if e == nil {
					return nil
				}
				step = e
			}
		}
	}
	// 末段省略时 cur 已停在 ']' 上
	if !p.curTokenIs(RBRACKET) {
		if !p.expectPeek(RBRACKET) {
			return nil
		}
	}
	if !isSlice {
		return &IndexExpression{Token: openTok, Left: left, Index: start}
	}
	return &SliceExpression{Token: openTok, Left: left, Start: start, Stop: stop, Step: step}
}

// parseAttributeExpression：obj.attr（进入时 curToken 是 '.'）
func (p *Parser) parseAttributeExpression(left Expression) Expression {
	exp := &AttributeExpression{Token: p.curToken, Object: left}
	if !p.expectPeek(IDENT) {
		return nil
	}
	exp.Attr = &Identifier{Token: p.curToken, Value: p.curToken.Literal}
	return exp
}

// parseIfExpressionInfix：Python 三元表达式 a if cond else b。
// 左操作数已经是 a；当前 curToken 是 IF。
func (p *Parser) parseIfExpressionInfix(left Expression) Expression {
	expr := &IfExpression{Token: p.curToken}
	p.nextToken() // 跳过 if
	expr.Condition = p.parseExpression(LOWEST)
	if expr.Condition == nil {
		return nil
	}
	if !p.expectPeek(ELSE) {
		return nil
	}
	p.nextToken()
	alt := p.parseExpression(LOWEST)
	if alt == nil {
		return nil
	}
	expr.Alternative = &BlockStatement{
		Token: p.curToken,
		Statements: []Statement{
			&ExpressionStatement{Token: p.curToken, Expression: alt},
		},
	}
	expr.Consequence = &BlockStatement{
		Token: p.curToken,
		Statements: []Statement{
			&ExpressionStatement{Token: p.curToken, Expression: left},
		},
	}
	return expr
}

// ----------------------------------------------------------------------------
// 索引赋值 AST 节点
// ----------------------------------------------------------------------------

// IndexAssignExpression 是把"arr[i] = x"建模成的特殊 AST 节点，
// 被 LetStatement.Value 持有，由 evaluator 识别执行。
type IndexAssignExpression struct {
	Token  Token
	Target Expression
	Index  Expression
	Value  Expression
}

func (iae *IndexAssignExpression) expressionNode()      {}
func (iae *IndexAssignExpression) TokenLiteral() string { return iae.Token.Literal }
func (iae *IndexAssignExpression) String() string {
	return fmt.Sprintf("%s[%s] = %s",
		iae.Target.String(), iae.Index.String(), iae.Value.String())
}