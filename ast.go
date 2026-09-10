package main

import "fmt"

// AST（抽象语法树）所有节点的统一接口。每个节点都要能返回它对应的字面量，
// 方便打印和调试。
type Node interface {
	TokenLiteral() string
	String() string
}

// Statement 是所有语句的接口。语句不产生值（赋值、if、while、def 都是）。
type Statement interface {
	Node
	statementNode()
}

// Expression 是所有表达式的接口。表达式会产生值。
type Expression interface {
	Node
	expressionNode()
}

// Program 是 AST 的根节点，承载一系列语句。
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	out := ""
	for _, s := range p.Statements {
		out += s.String()
	}
	return out
}

// ----------------------------------------------------------------------------
// 语句节点
// ----------------------------------------------------------------------------

// Let 语句：name = expr  （用 Let 而不是 Assign，跟 Monkey 风格保持一致；
// Python 的赋值不是表达式，所以也走语句）
type LetStatement struct {
	Token Token // token.LET... 实际上 token 是 NAME（IDENT）
	Name  *Identifier
	Value Expression
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	return fmt.Sprintf("%s = %s", ls.Name.String(), ls.Value.String())
}

// AugAssignStatement：复合赋值 x op= expr（目标也可以是 arr[i] / d[k]，
// 那时走 ExpressionStatement 里的 IndexAssignExpression 包装）。
type AugAssignStatement struct {
	Token    Token // 运算符 token（如 '+='）
	Target   Expression
	Operator string // 归一化后的运算符：+ - * / // %
	Value    Expression
}

func (as *AugAssignStatement) statementNode()       {}
func (as *AugAssignStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AugAssignStatement) String() string {
	return fmt.Sprintf("%s %s= %s", as.Target.String(), as.Operator, as.Value.String())
}

// UnpackAssignStatement：a, b = expr（序列解包赋值；RHS 可以是
// 隐式元组 a, b = 1, 2，parser 会包成 TupleLiteral）。
type UnpackAssignStatement struct {
	Token   Token // 首个目标 IDENT
	Targets []*Identifier
	Value   Expression
}

func (us *UnpackAssignStatement) statementNode()       {}
func (us *UnpackAssignStatement) TokenLiteral() string { return us.Token.Literal }
func (us *UnpackAssignStatement) String() string {
	names := []string{}
	for _, t := range us.Targets {
		names = append(names, t.String())
	}
	return fmt.Sprintf("%s = %s", joinStrings(names, ", "), us.Value.String())
}

// Return 语句：return expr
type ReturnStatement struct {
	Token       Token // 'return' token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	if rs.ReturnValue != nil {
		return fmt.Sprintf("return %s;", rs.ReturnValue.String())
	}
	return "return;"
}

// ExpressionStatement 表达式作为语句：x + 1、f(2) 等等
type ExpressionStatement struct {
	Token      Token // 表达式的第一个 token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// BlockStatement 用大括号语义包住一组语句：在 Python 里它对应 INDENT..DEDENT
// 之间的语句块。Evaluator 会拿到一个 BlockStatement 去逐条执行。
type BlockStatement struct {
	Token      Token // { 或 INDENT token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	out := ""
	for _, s := range bs.Statements {
		out += s.String()
	}
	return out
}

// IfStatement：if cond: block (elif cond: block)* (else: block)?
type IfStatement struct {
	Token       Token // 'if' token
	Condition   Expression
	Consequence *BlockStatement
	Elifs       []*ElifClause
	Alternative *BlockStatement // else
}

// ElifClause 一个 elif 分支（条件+块）。和 if 分开建模便于复用结构。
type ElifClause struct {
	Token       Token
	Condition   Expression
	Consequence *BlockStatement
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }
func (is *IfStatement) String() string {
	out := fmt.Sprintf("if %s %s", is.Condition.String(), is.Consequence.String())
	for _, e := range is.Elifs {
		out += fmt.Sprintf("elif %s %s", e.Condition.String(), e.Consequence.String())
	}
	if is.Alternative != nil {
		out += fmt.Sprintf("else %s", is.Alternative.String())
	}
	return out
}

// WhileStatement：while cond: block
type WhileStatement struct {
	Token     Token
	Condition Expression
	Body      *BlockStatement
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WhileStatement) String() string {
	return fmt.Sprintf("while %s %s", ws.Condition.String(), ws.Body.String())
}

// ForStatement：for x in iter: block 或 for k, v in iter: block
type ForStatement struct {
	Token   Token
	Targets []*Identifier
	Iter    Expression
	Body    *BlockStatement
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) String() string {
	names := []string{}
	for _, t := range fs.Targets {
		names = append(names, t.String())
	}
	return fmt.Sprintf("for %s in %s %s", joinStrings(names, ", "), fs.Iter.String(), fs.Body.String())
}

// Parameter 是函数形参:名字 + 可选默认值表达式。
type Parameter struct {
	Name    *Identifier
	Default Expression // nil 表示无默认值(必填参数)
}

func (p *Parameter) String() string {
	if p.Default != nil {
		return p.Name.Value + "=" + p.Default.String()
	}
	return p.Name.Value
}

// FunctionLiteral：def name(params): body
// （用 FunctionLiteral 而不是 FunctionStatement，更贴近表达式风格；
//   def 本身是语句，由 FunctionStatement 包住它）
type FunctionLiteral struct {
	Token      Token // 'def' token
	Name       *Identifier
	Parameters []*Parameter
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	params := []string{}
	for _, p := range fl.Parameters {
		params = append(params, p.String())
	}
	return fmt.Sprintf("def %s(%s) %s", fl.Name.String(), joinStrings(params, ", "), fl.Body.String())
}

// FunctionStatement def 语句（外层），因为 def 是语句而非表达式。
type FunctionStatement struct {
	Token      Token
	Name       *Identifier
	Parameters []*Parameter
	Body       *BlockStatement
}

func (fs *FunctionStatement) statementNode()       {}
func (fs *FunctionStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *FunctionStatement) String() string {
	params := []string{}
	for _, p := range fs.Parameters {
		params = append(params, p.String())
	}
	return fmt.Sprintf("def %s(%s) %s", fs.Name.String(), joinStrings(params, ", "), fs.Body.String())
}

// BreakStatement、ContinueStatement、PassStatement
type BreakStatement struct{ Token Token }

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return "break" }

type ContinueStatement struct{ Token Token }

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string       { return "continue" }

type PassStatement struct{ Token Token }

func (ps *PassStatement) statementNode()       {}
func (ps *PassStatement) TokenLiteral() string { return ps.Token.Literal }
func (ps *PassStatement) String() string       { return "pass" }

// TryStatement：try: block (except [Kind [as name]]: block)+
type TryStatement struct {
	Token    Token
	Body     *BlockStatement
	Handlers []*ExceptClause
}

func (ts *TryStatement) statementNode()       {}
func (ts *TryStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *TryStatement) String() string {
	out := fmt.Sprintf("try %s", ts.Body.String())
	for _, h := range ts.Handlers {
		out += fmt.Sprintf("except %s %s", h.Kind, h.Body.String())
	}
	return out
}

// ExceptClause 一个 except 分支;Kind 为空表示裸 except,BindName 为 as 绑定的名字
type ExceptClause struct {
	Token    Token
	Kind     string
	BindName string
	Body     *BlockStatement
}

// RaiseStatement：raise expr（抛出异常,expr 通常是对异常类的调用）
type RaiseStatement struct {
	Token Token
	Value Expression
}

func (rs *RaiseStatement) statementNode()       {}
func (rs *RaiseStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *RaiseStatement) String() string {
	if rs.Value != nil {
		return fmt.Sprintf("raise %s", rs.Value.String())
	}
	return "raise"
}

// ----------------------------------------------------------------------------
// 表达式节点
// ----------------------------------------------------------------------------

type Identifier struct {
	Token Token // token.IDENT
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral
type IntegerLiteral struct {
	Token Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral
type FloatLiteral struct {
	Token Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral
type StringLiteral struct {
	Token Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return sl.Token.Literal }

// BooleanLiteral（独立节点，区分于普通标识符）
type BooleanLiteral struct {
	Token Token
	Value bool
}

func (b *BooleanLiteral) expressionNode()      {}
func (b *BooleanLiteral) TokenLiteral() string { return b.Token.Literal }
func (b *BooleanLiteral) String() string       { return b.Token.Literal }

// NoneLiteral
type NoneLiteral struct{ Token Token }

func (n *NoneLiteral) expressionNode()      {}
func (n *NoneLiteral) TokenLiteral() string { return n.Token.Literal }
func (n *NoneLiteral) String() string       { return "None" }

// PrefixExpression：-x, not x
type PrefixExpression struct {
	Token    Token // 前缀运算符 token, e.g. !
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	return fmt.Sprintf("(%s%s)", pe.Operator, pe.Right.String())
}

// InfixExpression：a + b, a == b
type InfixExpression struct {
	Token    Token // 运算符 token, e.g. +
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	return fmt.Sprintf("(%s %s %s)", ie.Left.String(), ie.Operator, ie.Right.String())
}

// IfExpression：value_if_true if cond else value_if_false（Python 的三元表达式）
type IfExpression struct {
	Token       Token // 'if' token of ternary
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	return fmt.Sprintf("(%s if %s else %s)",
		ie.Consequence.String(), ie.Condition.String(), ie.Alternative.String())
}

// CallExpression：f(a, b) 或 f(a, key=val)
type CallExpression struct {
	Token     Token // '(' token
	Function  Expression // Identifier 或 FunctionLiteral
	Arguments []Expression
	ArgNames  []string // 与 Arguments 对齐;空串表示位置参数,非空表示关键字名
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	args := []string{}
	for i, a := range ce.Arguments {
		if ce.ArgNames != nil && ce.ArgNames[i] != "" {
			args = append(args, ce.ArgNames[i]+"="+a.String())
		} else {
			args = append(args, a.String())
		}
	}
	return fmt.Sprintf("%s(%s)", ce.Function.String(), joinStrings(args, ", "))
}

// ArrayLiteral：[1, 2, 3]
type ArrayLiteral struct {
	Token    Token // '['
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	elems := []string{}
	for _, e := range al.Elements {
		elems = append(elems, e.String())
	}
	return fmt.Sprintf("[%s]", joinStrings(elems, ", "))
}

// DictLiteral：{"a": 1, "b": 2} 或 {}
type DictLiteral struct {
	Token  Token // '{'
	Keys   []Expression
	Values []Expression
}

func (dl *DictLiteral) expressionNode()      {}
func (dl *DictLiteral) TokenLiteral() string { return dl.Token.Literal }
func (dl *DictLiteral) String() string {
	parts := []string{}
	for i := range dl.Keys {
		parts = append(parts, dl.Keys[i].String()+": "+dl.Values[i].String())
	}
	return fmt.Sprintf("{%s}", joinStrings(parts, ", "))
}

// AttributeExpression：obj.attr（方法访问；未来也用于对象属性）
type AttributeExpression struct {
	Token  Token // '.'
	Object Expression
	Attr   *Identifier
}

func (ae *AttributeExpression) expressionNode()      {}
func (ae *AttributeExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AttributeExpression) String() string {
	return fmt.Sprintf("%s.%s", ae.Object.String(), ae.Attr.Value)
}

// IndexExpression：arr[i]
type IndexExpression struct {
	Token Token // '['
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	return fmt.Sprintf("(%s[%s])", ie.Left.String(), ie.Index.String())
}

// SliceExpression：seq[start:stop:step]，三段均可省略
type SliceExpression struct {
	Token Token // '['
	Left  Expression
	Start Expression // 可为 nil
	Stop  Expression // 可为 nil
	Step  Expression // 可为 nil
}

func (se *SliceExpression) expressionNode()      {}
func (se *SliceExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SliceExpression) String() string {
	part := func(e Expression) string {
		if e == nil {
			return ""
		}
		return e.String()
	}
	return fmt.Sprintf("(%s[%s:%s:%s])",
		se.Left.String(), part(se.Start), part(se.Stop), part(se.Step))
}

// TupleLiteral：(1, 2) / (1,) / ()，也承载解包赋值的隐式元组 RHS
type TupleLiteral struct {
	Token    Token // '('
	Elements []Expression
}

func (tl *TupleLiteral) expressionNode()      {}
func (tl *TupleLiteral) TokenLiteral() string { return tl.Token.Literal }
func (tl *TupleLiteral) String() string {
	elems := []string{}
	for _, e := range tl.Elements {
		elems = append(elems, e.String())
	}
	if len(elems) == 1 {
		return "(" + elems[0] + ",)"
	}
	return "(" + joinStrings(elems, ", ") + ")"
}

// AssignExpression（在表达式位置也能赋值，比如 arr[0] = 9）。
// Python 实际是语句，但作为表达式在 [i]=... 之类的位置处理更顺手，
// 所以我们仍保留为语句风格（见 LetStatement），Index 写入作为特例处理。
// 这里仍然建模成表达式求值返回赋值后的值，方便 a = b = 1 之类的链式（虽然暂不支持）。

// 辅助函数
func joinStrings(items []string, sep string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}