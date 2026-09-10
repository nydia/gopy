package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
)

// Eval 把 AST 节点求值为一个 Object。
// 错误以 *Error 对象形式冒泡到顶层，由调用者负责打印和退出。
func Eval(node Node, env *Environment) Object {
	if os.Getenv("GOPY_DEBUG") != "" {
		switch n := node.(type) {
		case *ExpressionStatement:
			if call, ok := n.Expression.(*CallExpression); ok {
				if id, ok := call.Function.(*Identifier); ok && id.Value == "print" {
					fmt.Fprintf(os.Stderr, "[Eval print] %s\n", n.String())
				}
			}
		}
	}
	switch node := node.(type) {

	// ------------------------------------------------------------------------
	// 语句
	// ------------------------------------------------------------------------
	case *Program:
		return evalProgram(node.Statements, env)

	case *ExpressionStatement:
		if node.Expression == nil {
			return NULL
		}
		return Eval(node.Expression, env)

	case *LetStatement:
		val := Eval(node.Value, env)
		if IsError(val) {
			return val
		}
		if env.IsConst(node.Name.Value) {
			return NewError("cannot reassign constant '%s'", node.Name.Value)
		}
		env.Set(node.Name.Value, val)
		return NULL

	case *AugAssignStatement:
		// x op= expr：读旧值 → 用中缀求值 → 写回
		id, ok := node.Target.(*Identifier)
		if !ok {
			return NewError("invalid augmented assignment target: %s", node.Target.String())
		}
		if env.IsConst(id.Value) {
			return NewError("cannot reassign constant '%s'", id.Value)
		}
		cur := Eval(node.Target, env)
		if IsError(cur) {
			return cur
		}
		val := Eval(node.Value, env)
		if IsError(val) {
			return val
		}
		result := evalInfixExpression(node.Operator, cur, val)
		if IsError(result) {
			return result
		}
		env.Set(id.Value, result)
		return NULL

	case *ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if IsError(val) {
			return val
		}
		return &ReturnValue{Value: val}

	case *BlockStatement:
		return evalBlockStatements(node.Statements, env)

	case *IfStatement:
		return evalIfStatement(node, env)

	case *WhileStatement:
		return evalWhileStatement(node, env)

	case *ForStatement:
		return evalForStatement(node, env)

	case *FunctionStatement:
		// 默认值在 def 时求值一次（CPython 语义,可变默认值跨调用共享）
		defaults := make([]Object, len(node.Parameters))
		for i, p := range node.Parameters {
			if p.Default != nil {
				v := Eval(p.Default, env)
				if IsError(v) {
					return v
				}
				defaults[i] = v
			}
		}
		fn := &Function{
			Name:       node.Name.Value,
			Parameters: node.Parameters,
			Defaults:   defaults,
			Body:       node.Body,
			Env:        env,
		}
		env.SetConst(node.Name.Value, fn)
		return NULL

	case *BreakStatement:
		return &Break{}
	case *ContinueStatement:
		return &Continue{}
	case *PassStatement:
		return NULL
	case *TryStatement:
		return evalTryStatement(node, env)
	case *RaiseStatement:
		if node.Value == nil {
			// bare raise:没有活动异常可重抛
			return NewErrorWithKind("RuntimeError", "No active exception to re-raise")
		}
		val := Eval(node.Value, env)
		if IsError(val) {
			return val
		}
		return exceptionToError(val)

	// ------------------------------------------------------------------------
	// 表达式
	// ------------------------------------------------------------------------
	case *Identifier:
		return evalIdentifier(node, env)

	case *IntegerLiteral:
		return &Integer{Value: node.Value}

	case *FloatLiteral:
		return &Float{Value: node.Value}

	case *StringLiteral:
		return &String{Value: node.Value}

	case *BooleanLiteral:
		return NativeBoolToBooleanObject(node.Value)

	case *NoneLiteral:
		return NULL

	case *PrefixExpression:
		right := Eval(node.Right, env)
		if IsError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *InfixExpression:
		// and/or 短路求值并返回操作数本身（CPython 语义），
		// 不能先对两侧求值，所以要在进入通用求值前拦截。
		if node.Operator == "and" || node.Operator == "or" {
			return evalLogicalExpression(node, env)
		}
		left := Eval(node.Left, env)
		if IsError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if IsError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)

	case *IfExpression:
		return evalIfExpression(node, env)

	case *CallExpression:
		return evalCallExpression(node, env)

	case *ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && IsError(elements[0]) {
			return elements[0]
		}
		return &Array{Elements: elements}

	case *DictLiteral:
		dict := NewDict()
		for i := range node.Keys {
			k := Eval(node.Keys[i], env)
			if IsError(k) {
				return k
			}
			v := Eval(node.Values[i], env)
			if IsError(v) {
				return v
			}
			if !dict.Set(k, v) {
				return NewError("unhashable type: '%s' as dict key", pyTypeName(k))
			}
		}
		return dict

	case *TupleLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && IsError(elements[0]) {
			return elements[0]
		}
		return &Tuple{Elements: elements}

	case *UnpackAssignStatement:
		val := Eval(node.Value, env)
		if IsError(val) {
			return val
		}
		if err := unpackInto(node.Targets, val, env); err != nil {
			return err
		}
		return NULL

	case *IndexExpression:
		left := Eval(node.Left, env)
		if IsError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if IsError(index) {
			return index
		}
		return evalIndexExpression(left, index)

	case *SliceExpression:
		left := Eval(node.Left, env)
		if IsError(left) {
			return left
		}
		evalPart := func(e Expression) Object {
			if e == nil {
				return nil
			}
			v := Eval(e, env)
			if IsError(v) {
				return v
			}
			return v
		}
		start := evalPart(node.Start)
		if IsError(start) {
			return start
		}
		stop := evalPart(node.Stop)
		if IsError(stop) {
			return stop
		}
		step := evalPart(node.Step)
		if IsError(step) {
			return step
		}
		return evalSliceExpression(left, start, stop, step)

	case *IndexAssignExpression:
		// arr[i] = x：把 target 求值（要求是列表），再求 index 和 value，写入。
		target := Eval(node.Target, env)
		if IsError(target) {
			return target
		}
		idx := Eval(node.Index, env)
		if IsError(idx) {
			return idx
		}
		val := Eval(node.Value, env)
		if IsError(val) {
			return val
		}
		return evalIndexAssign(target, idx, val)

	case *AttributeExpression:
		return evalAttributeExpression(node, env)
	}

	return NewError("unknown node type: %T", node)
}

// ----------------------------------------------------------------------------
// 程序 / 块
// ----------------------------------------------------------------------------

func evalProgram(stmts []Statement, env *Environment) Object {
	var result Object = NULL
	for _, stmt := range stmts {
		result = Eval(stmt, env)
		switch r := result.(type) {
		case *ReturnValue:
			return r.Value
		case *Error:
			return r
		case *Break, *Continue:
			// 顶层遇到 break/continue 是错误
			return NewError("'%s' outside loop", r.Inspect())
		}
	}
	return result
}

func evalBlockStatements(stmts []Statement, env *Environment) Object {
	var result Object = NULL
	for _, stmt := range stmts {
		result = Eval(stmt, env)
		if result != nil {
			rt := result.Type()
			if rt == RETURN_VALUE_OBJ || rt == ERROR_OBJ || rt == BREAK_OBJ || rt == CONTINUE_OBJ {
				return result
			}
		}
	}
	return result
}

// ----------------------------------------------------------------------------
// 控制流
// ----------------------------------------------------------------------------

// evalLogicalExpression 实现 and / or 的短路语义：
// a and b → a 为假返回 a，否则返回 b；a or b → a 为真返回 a，否则返回 b。
func evalLogicalExpression(node *InfixExpression, env *Environment) Object {
	left := Eval(node.Left, env)
	if IsError(left) {
		return left
	}
	if node.Operator == "and" {
		if !truthy(left) {
			return left
		}
		return Eval(node.Right, env)
	}
	if truthy(left) {
		return left
	}
	return Eval(node.Right, env)
}

// truthy：Python 的真值判断——除了 False/None/0/""/[]/{} 之外都算真。
func truthy(obj Object) bool {
	switch o := obj.(type) {
	case *Boolean:
		return o.Value
	case *Null:
		return false
	case *Integer:
		return o.Value != 0
	case *Float:
		return o.Value != 0
	case *String:
		return o.Value != ""
	case *Array:
		return len(o.Elements) > 0
	case *Dict:
		return o.Len() > 0
	case *Tuple:
		return len(o.Elements) > 0
	}
	return true
}

// unpackInto 把可迭代对象 val 的元素按序绑定到 targets。
// 数量不匹配时报 CPython 风格错误；返回 nil 表示成功。
func unpackInto(targets []*Identifier, val Object, env *Environment) Object {
	var items []Object
	switch v := val.(type) {
	case *Array:
		items = v.Elements
	case *Tuple:
		items = v.Elements
	case *String:
		items = []Object{}
		for _, r := range v.Value {
			items = append(items, &String{Value: string(r)})
		}
	default:
		return NewError("cannot unpack non-iterable %s object", pyTypeName(val))
	}
	if len(items) < len(targets) {
		return NewError("not enough values to unpack (expected %d, got %d)",
			len(targets), len(items))
	}
	if len(items) > len(targets) {
		return NewError("too many values to unpack (expected %d)", len(targets))
	}
	for i, t := range targets {
		env.Set(t.Value, items[i])
	}
	return nil
}

// evalTryStatement 执行 try 块:结果若是 Error 且能匹配某个 except,
// 则绑定消息(可选)并执行 handler;否则原样冒泡。
// break/continue/return 信号不是 Error,原样放行给外层。
func evalTryStatement(stmt *TryStatement, env *Environment) Object {
	result := Eval(stmt.Body, env)
	errObj, ok := result.(*Error)
	if !ok {
		return result
	}
	for _, h := range stmt.Handlers {
		if h.Kind == "" || h.Kind == errObj.Kind {
			if h.BindName != "" {
				env.Set(h.BindName, &String{Value: errObj.Message})
			}
			return Eval(h.Body, env)
		}
	}
	return result
}

// exceptionToError 把 raise 的值转换成异常信号(Error)。
func exceptionToError(val Object) Object {
	switch v := val.(type) {
	case *ExceptionInstance:
		return &Error{Kind: v.KindName, Message: v.Message}
	case *String:
		return &Error{Message: v.Value}
	case *Error:
		return v
	}
	return &Error{Message: val.Inspect()}
}

func evalIfStatement(stmt *IfStatement, env *Environment) Object {
	cond := Eval(stmt.Condition, env)
	if IsError(cond) {
		return cond
	}
	if truthy(cond) {
		return Eval(stmt.Consequence, env)
	}
	for _, e := range stmt.Elifs {
		c := Eval(e.Condition, env)
		if IsError(c) {
			return c
		}
		if truthy(c) {
			return Eval(e.Consequence, env)
		}
	}
	if stmt.Alternative != nil {
		return Eval(stmt.Alternative, env)
	}
	return NULL
}

func evalIfExpression(expr *IfExpression, env *Environment) Object {
	cond := Eval(expr.Condition, env)
	if IsError(cond) {
		return cond
	}
	if truthy(cond) {
		return Eval(expr.Consequence, env)
	}
	return Eval(expr.Alternative, env)
}

func evalWhileStatement(stmt *WhileStatement, env *Environment) Object {
	var result Object = NULL
	for {
		cond := Eval(stmt.Condition, env)
		if IsError(cond) {
			return cond
		}
		if !truthy(cond) {
			return result
		}
		result = Eval(stmt.Body, env)
		if result == nil {
			continue
		}
		switch result.Type() {
		case BREAK_OBJ:
			return NULL
		case CONTINUE_OBJ:
			continue
		case RETURN_VALUE_OBJ, ERROR_OBJ:
			return result
		}
	}
}

func evalForStatement(stmt *ForStatement, env *Environment) Object {
	iterObj := Eval(stmt.Iter, env)
	if IsError(iterObj) {
		return iterObj
	}
	// 可迭代对象统一展开成元素序列：list/tuple → 元素，dict → 键（插入序），
	// str → 逐字符（按 rune，支持中文）。
	var items []Object
	switch it := iterObj.(type) {
	case *Array:
		items = it.Elements
	case *Tuple:
		items = it.Elements
	case *Dict:
		items = it.Keys()
	case *String:
		items = []Object{}
		for _, r := range it.Value {
			items = append(items, &String{Value: string(r)})
		}
	default:
		return NewError("for-in target must be iterable (got %s)", iterObj.Type())
	}
	var result Object = NULL
	for _, item := range items {
		// 把循环变量直接绑到外层 env（Python 的 for 变量实际会泄漏，
		// 而 body 内的赋值也必须影响外层，所以这里共享 env 即可）。
		if len(stmt.Targets) == 1 {
			env.Set(stmt.Targets[0].Value, item)
		} else {
			// for k, v in pairs: 每个元素再解包到多个目标
			if err := unpackInto(stmt.Targets, item, env); err != nil {
				return err
			}
		}
		result = Eval(stmt.Body, env)
		if result == nil {
			continue
		}
		switch result.Type() {
		case BREAK_OBJ:
			return NULL
		case CONTINUE_OBJ:
			continue
		case RETURN_VALUE_OBJ, ERROR_OBJ:
			return result
		}
	}
	return result
}

// ----------------------------------------------------------------------------
// 标识符 / 前缀 / 中缀
// ----------------------------------------------------------------------------

func evalIdentifier(node *Identifier, env *Environment) Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}
	if b, ok := builtins[node.Value]; ok {
		return b
	}
	return NewErrorWithKind("NameError", "identifier not found: %s", node.Value)
}

func evalPrefixExpression(operator string, right Object) Object {
	switch operator {
	case "-":
		return evalMinusPrefixOperator(right)
	case "not":
		return NativeBoolToBooleanObject(!truthy(right))
	}
	return NewError("unknown operator: %s%s", operator, right.Type())
}

func evalMinusPrefixOperator(right Object) Object {
	switch v := right.(type) {
	case *Integer:
		return &Integer{Value: -v.Value}
	case *Float:
		return &Float{Value: -v.Value}
	}
	return NewError("unsupported operand type for unary -: %s", right.Type())
}

func evalInfixExpression(operator string, left, right Object) Object {
	switch {
	// 相等比较跨类型不报错：1 == "1" → False（对齐 CPython）
	case operator == "==":
		return NativeBoolToBooleanObject(objectsEqual(left, right))
	case operator == "!=":
		return NativeBoolToBooleanObject(!objectsEqual(left, right))
	case operator == "in" || operator == "not in":
		return evalMembership(operator, left, right)
	case left.Type() == INTEGER_OBJ && right.Type() == INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == FLOAT_OBJ || right.Type() == FLOAT_OBJ:
		return evalFloatInfixExpression(operator, left, right)
	case left.Type() == STRING_OBJ && right.Type() == STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case operator == "*":
		// "ab" * 3 或 3 * "ab"（字符串重复）；[1] * 3 列表重复
		if left.Type() == ARRAY_OBJ && right.Type() == INTEGER_OBJ {
			arr := left.(*Array)
			n := right.(*Integer).Value
			return &Array{Elements: repeatElements(arr.Elements, n)}
		}
		if left.Type() == INTEGER_OBJ && right.Type() == ARRAY_OBJ {
			arr := right.(*Array)
			n := left.(*Integer).Value
			return &Array{Elements: repeatElements(arr.Elements, n)}
		}
		var s *String
		var n *Integer
		if v, ok := left.(*String); ok {
			s = v
			n, _ = right.(*Integer)
		} else {
			s, _ = right.(*String)
			n, _ = left.(*Integer)
		}
		if s != nil && n != nil {
			return &String{Value: repeatString(s.Value, n.Value)}
		}
		bad := right
		if s == nil {
			bad = left
		}
		return NewError("can't multiply sequence by non-int of type '%s'", pyTypeName(bad))
	case operator == "+" && left.Type() == ARRAY_OBJ && right.Type() == ARRAY_OBJ:
		// 列表拼接：返回新列表
		l := left.(*Array)
		r := right.(*Array)
		merged := make([]Object, 0, len(l.Elements)+len(r.Elements))
		merged = append(merged, l.Elements...)
		merged = append(merged, r.Elements...)
		return &Array{Elements: merged}
	case left.Type() != right.Type():
		return NewError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	case left.Type() == BOOLEAN_OBJ && right.Type() == BOOLEAN_OBJ:
		return evalBooleanInfixExpression(operator, left, right)
	}
	return NewError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
}

// repeatString 实现字符串重复；n <= 0 得到空串（与 CPython 一致）。
func repeatString(s string, n int64) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, int(n))
}

// repeatElements 实现序列重复；n <= 0 得到空列表。
func repeatElements(elems []Object, n int64) []Object {
	if n <= 0 {
		return []Object{}
	}
	out := make([]Object, 0, int64(len(elems))*n)
	for i := int64(0); i < n; i++ {
		out = append(out, elems...)
	}
	return out
}

// objectsEqual 判断两个对象的"Python == 语义"是否相等。
// 数值跨 int/float 比较，bool 与 int 可比（True == 1），
// 容器递归逐元素比较；类型不可比时一律 False。
func objectsEqual(a, b Object) bool {
	switch av := a.(type) {
	case *Integer:
		switch bv := b.(type) {
		case *Integer:
			return av.Value == bv.Value
		case *Float:
			return float64(av.Value) == bv.Value
		case *Boolean:
			return av.Value == boolToInt64(bv.Value)
		}
		return false
	case *Float:
		switch bv := b.(type) {
		case *Float:
			return av.Value == bv.Value
		case *Integer:
			return av.Value == float64(bv.Value)
		}
		return false
	case *Boolean:
		switch bv := b.(type) {
		case *Boolean:
			return av.Value == bv.Value
		case *Integer:
			return boolToInt64(av.Value) == bv.Value
		}
		return false
	case *String:
		bv, ok := b.(*String)
		return ok && av.Value == bv.Value
	case *Null:
		_, ok := b.(*Null)
		return ok
	case *Array:
		bv, ok := b.(*Array)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i, e := range av.Elements {
			if !objectsEqual(e, bv.Elements[i]) {
				return false
			}
		}
		return true
	case *Tuple:
		bv, ok := b.(*Tuple)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i, e := range av.Elements {
			if !objectsEqual(e, bv.Elements[i]) {
				return false
			}
		}
		return true
	case *Dict:
		bv, ok := b.(*Dict)
		if !ok || av.Len() != bv.Len() {
			return false
		}
		for _, k := range av.keys {
			bp, ok := bv.pairs[k]
			if !ok || !objectsEqual(av.pairs[k].Value, bp.Value) {
				return false
			}
		}
		return true
	}
	return a == b
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// evalMembership 实现 in / not in 成员运算。
// str 要求左侧也是 str；list 逐元素比较；dict 按键判断。
func evalMembership(operator string, left, right Object) Object {
	found := false
	switch container := right.(type) {
	case *String:
		s, ok := left.(*String)
		if !ok {
			return NewError("'in <string>' requires string as left operand, not %s",
				pyTypeName(left))
		}
		found = strings.Contains(container.Value, s.Value)
	case *Array:
		for _, e := range container.Elements {
			if objectsEqual(e, left) {
				found = true
				break
			}
		}
	case *Tuple:
		for _, e := range container.Elements {
			if objectsEqual(e, left) {
				found = true
				break
			}
		}
	case *Dict:
		_, ok := container.Get(left)
		found = ok
	default:
		return NewError("argument of type '%s' is not iterable", right.Type())
	}
	if operator == "not in" {
		found = !found
	}
	return NativeBoolToBooleanObject(found)
}

func evalIntegerInfixExpression(operator string, left, right Object) Object {
	l := left.(*Integer).Value
	r := right.(*Integer).Value
	switch operator {
	case "+":
		return &Integer{Value: l + r}
	case "-":
		return &Integer{Value: l - r}
	case "*":
		return &Integer{Value: l * r}
	case "/":
		if r == 0 {
			return NewErrorWithKind("ZeroDivisionError", "division by zero")
		}
		// Python 的 / 是真除法：int/int 结果总是 float
		return &Float{Value: float64(l) / float64(r)}
	case "//":
		if r == 0 {
			return NewErrorWithKind("ZeroDivisionError", "integer division or modulo by zero")
		}
		// 向负无穷取整（Go 的 / 向零截断，负数时需修正）
		q := l / r
		if l%r != 0 && (l < 0) != (r < 0) {
			q--
		}
		return &Integer{Value: q}
	case "%":
		if r == 0 {
			return NewErrorWithKind("ZeroDivisionError", "modulo by zero")
		}
		// Python 的模运算结果符号跟随除数（Go 的 % 符号跟随被除数）
		m := l % r
		if m != 0 && (m < 0) != (r < 0) {
			m += r
		}
		return &Integer{Value: m}
	case "==":
		return NativeBoolToBooleanObject(l == r)
	case "!=":
		return NativeBoolToBooleanObject(l != r)
	case "<":
		return NativeBoolToBooleanObject(l < r)
	case ">":
		return NativeBoolToBooleanObject(l > r)
	case "<=":
		return NativeBoolToBooleanObject(l <= r)
	case ">=":
		return NativeBoolToBooleanObject(l >= r)
	}
	return NewError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
}

func evalFloatInfixExpression(operator string, left, right Object) Object {
	lf := toFloat(left)
	rf := toFloat(right)
	switch operator {
	case "+":
		return &Float{Value: lf + rf}
	case "-":
		return &Float{Value: lf - rf}
	case "*":
		return &Float{Value: lf * rf}
	case "/":
		if rf == 0 {
			return NewErrorWithKind("ZeroDivisionError", "float division by zero")
		}
		return &Float{Value: lf / rf}
	case "%":
		if rf == 0 {
			return NewErrorWithKind("ZeroDivisionError", "float modulo")
		}
		// 模仿 Python：x % y = x - math.floor(x/y) * y
		q := fFloor(lf / rf)
		return &Float{Value: lf - q*rf}
	case "//":
		if rf == 0 {
			return NewErrorWithKind("ZeroDivisionError", "float floor division by zero")
		}
		return &Float{Value: fFloor(lf / rf)}
	case "==":
		return NativeBoolToBooleanObject(lf == rf)
	case "!=":
		return NativeBoolToBooleanObject(lf != rf)
	case "<":
		return NativeBoolToBooleanObject(lf < rf)
	case ">":
		return NativeBoolToBooleanObject(lf > rf)
	case "<=":
		return NativeBoolToBooleanObject(lf <= rf)
	case ">=":
		return NativeBoolToBooleanObject(lf >= rf)
	}
	return NewError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
}

func toFloat(o Object) float64 {
	switch v := o.(type) {
	case *Integer:
		return float64(v.Value)
	case *Float:
		return v.Value
	}
	return 0
}

func fFloor(x float64) float64 {
	// floor：向负无穷取整。int64(x) 向零截断，只要截断值大于 x 就退一步。
	i := int64(x)
	if float64(i) > x {
		i--
	}
	return float64(i)
}

func evalStringInfixExpression(operator string, left, right Object) Object {
	l := left.(*String).Value
	r := right.(*String).Value
	switch operator {
	case "+":
		return &String{Value: l + r}
	case "<":
		return NativeBoolToBooleanObject(l < r)
	case ">":
		return NativeBoolToBooleanObject(l > r)
	case "<=":
		return NativeBoolToBooleanObject(l <= r)
	case ">=":
		return NativeBoolToBooleanObject(l >= r)
	}
	return NewError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
}

func evalBooleanInfixExpression(operator string, left, right Object) Object {
	l := left.(*Boolean).Value
	r := right.(*Boolean).Value
	switch operator {
	case "and":
		return NativeBoolToBooleanObject(l && r)
	case "or":
		return NativeBoolToBooleanObject(l || r)
	case "==":
		return NativeBoolToBooleanObject(l == r)
	case "!=":
		return NativeBoolToBooleanObject(l != r)
	}
	return NewError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
}

// ----------------------------------------------------------------------------
// 函数调用 / 下标
// ----------------------------------------------------------------------------

func evalCallExpression(node *CallExpression, env *Environment) Object {
	function := Eval(node.Function, env)
	if IsError(function) {
		return function
	}
	// 位置参数与关键字参数分开收集
	args := []Object{}
	kwargs := map[string]Object{}
	for i, e := range node.Arguments {
		v := Eval(e, env)
		if IsError(v) {
			return v
		}
		if name := node.ArgNames[i]; name != "" {
			kwargs[name] = v
		} else {
			args = append(args, v)
		}
	}
	return applyFunction(function, args, kwargs)
}

func evalExpressions(exps []Expression, env *Environment) []Object {
	result := make([]Object, 0, len(exps))
	for _, e := range exps {
		evaluated := Eval(e, env)
		if IsError(evaluated) {
			return []Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

func applyFunction(fn Object, args []Object, kwargs map[string]Object) Object {
	switch fn := fn.(type) {
	case *Function:
		n := len(fn.Parameters)
		// 位置参数按序填充
		if len(args) > n {
			return NewError("function %s takes at most %d positional argument (%d given)",
				fn.Name, n, len(args))
		}
		values := make([]Object, n)
		provided := make([]bool, n)
		for i, a := range args {
			values[i] = a
			provided[i] = true
		}
		// 关键字参数按名字填充
		for name, v := range kwargs {
			idx := -1
			for i, p := range fn.Parameters {
				if p.Name.Value == name {
					idx = i
					break
				}
			}
			if idx == -1 {
				return NewError("function %s got an unexpected keyword argument '%s'",
					fn.Name, name)
			}
			if provided[idx] {
				return NewError("function %s got multiple values for argument '%s'",
					fn.Name, name)
			}
			values[idx] = v
			provided[idx] = true
		}
		// 缺省填默认值(已按 def 时求值缓存)
		for i, p := range fn.Parameters {
			if !provided[i] {
				if fn.Defaults[i] != nil {
					values[i] = fn.Defaults[i]
				} else {
					return NewError("function %s missing required argument '%s'",
						fn.Name, p.Name.Value)
				}
			}
		}
		// 构造新作用域
		callEnv := NewEnclosedEnvironment(fn.Env)
		for i, p := range fn.Parameters {
			callEnv.Set(p.Name.Value, values[i])
		}
		evaluated := Eval(fn.Body, callEnv)
		if rv, ok := evaluated.(*ReturnValue); ok {
			return rv.Value
		}
		return evaluated

	case *Builtin:
		if len(kwargs) > 0 {
			return NewError("builtin function takes no keyword arguments")
		}
		return fn.Fn(args...)

	case *BoundMethod:
		if len(kwargs) > 0 {
			return NewError("method %s() takes no keyword arguments", fn.Name)
		}
		return fn.Fn(append([]Object{fn.Receiver}, args...)...)

	case *ExceptionClass:
		if len(kwargs) > 0 || len(args) > 1 {
			return NewError("%s() takes at most 1 argument (%d given)", fn.Name, len(args))
		}
		if len(args) == 1 {
			return &ExceptionInstance{KindName: fn.Name, Message: args[0].Inspect()}
		}
		return &ExceptionInstance{KindName: fn.Name}

	default:
		return NewError("not a function: %s", fn.Type())
	}
}

// ----------------------------------------------------------------------------
// 方法调用：obj.attr 求值与方法表
// ----------------------------------------------------------------------------

// evalAttributeExpression 求值 obj.attr：按接收者类型查方法表，
// 返回绑定接收者的 BoundMethod（方法是一等值，可赋给变量后调用）。
func evalAttributeExpression(node *AttributeExpression, env *Environment) Object {
	obj := Eval(node.Object, env)
	if IsError(obj) {
		return obj
	}
	var table map[string]*Builtin
	switch obj.(type) {
	case *Array:
		table = listMethods
	case *Dict:
		table = dictMethods
	case *String:
		table = strMethods
	default:
		return NewError("'%s' object has no attribute '%s'", pyTypeName(obj), node.Attr.Value)
	}
	m, ok := table[node.Attr.Value]
	if !ok {
		return NewError("'%s' object has no attribute '%s'", pyTypeName(obj), node.Attr.Value)
	}
	return &BoundMethod{Receiver: obj, Name: node.Attr.Value, Fn: m.Fn}
}

// compareValues 按 Python 的 < 语义比较两个对象，返回 -1/0/1。
// 类型不可比较时返回错误对象。
func compareValues(a, b Object) (int, Object) {
	switch av := a.(type) {
	case *Integer:
		switch bv := b.(type) {
		case *Integer:
			switch {
			case av.Value < bv.Value:
				return -1, nil
			case av.Value > bv.Value:
				return 1, nil
			}
			return 0, nil
		case *Float:
			return compareFloat(float64(av.Value), bv.Value)
		}
	case *Float:
		switch bv := b.(type) {
		case *Float:
			return compareFloat(av.Value, bv.Value)
		case *Integer:
			return compareFloat(av.Value, float64(bv.Value))
		}
	case *String:
		bv, ok := b.(*String)
		if ok {
			return strings.Compare(av.Value, bv.Value), nil
		}
	}
	return 0, NewErrorWithKind("TypeError", "'<' not supported between instances of '%s' and '%s'",
		pyTypeName(a), pyTypeName(b))
}

func compareFloat(a, b float64) (int, Object) {
	switch {
	case a < b:
		return -1, nil
	case a > b:
		return 1, nil
	}
	return 0, nil
}

// ----------------------------------------------------------------------------
// list / dict 方法表。调用时 args[0] 是接收者。
// ----------------------------------------------------------------------------

var listMethods = map[string]*Builtin{
	"append": {Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return NewError("append() takes exactly one argument (%d given)", len(args)-1)
		}
		list := args[0].(*Array)
		list.Elements = append(list.Elements, args[1])
		return NULL
	}},
	"pop": {Fn: func(args ...Object) Object {
		list := args[0].(*Array)
		n := len(list.Elements)
		if n == 0 {
			return NewErrorWithKind("IndexError", "pop from empty list")
		}
		idx := int64(n - 1)
		if len(args) == 2 {
			i, ok := args[1].(*Integer)
			if !ok {
				return NewError("list index must be integer, not %s", args[1].Type())
			}
			idx = i.Value
			if idx < 0 {
				idx = int64(n) + idx
			}
		} else if len(args) > 2 {
			return NewError("pop() takes at most one argument (%d given)", len(args)-1)
		}
		if idx < 0 || idx >= int64(n) {
			return NewErrorWithKind("IndexError", "pop index out of range")
		}
		val := list.Elements[idx]
		list.Elements = append(list.Elements[:idx], list.Elements[idx+1:]...)
		return val
	}},
	"insert": {Fn: func(args ...Object) Object {
		if len(args) != 3 {
			return NewError("insert() takes exactly two arguments (%d given)", len(args)-1)
		}
		list := args[0].(*Array)
		i, ok := args[1].(*Integer)
		if !ok {
			return NewError("list index must be integer, not %s", args[1].Type())
		}
		n := int64(len(list.Elements))
		idx := i.Value
		if idx < 0 {
			idx = n + idx
		}
		if idx < 0 {
			idx = 0
		}
		if idx > n {
			idx = n
		}
		list.Elements = append(list.Elements, nil)
		copy(list.Elements[idx+1:], list.Elements[idx:])
		list.Elements[idx] = args[2]
		return NULL
	}},
	"extend": {Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return NewError("extend() takes exactly one argument (%d given)", len(args)-1)
		}
		list := args[0].(*Array)
		other, ok := args[1].(*Array)
		if !ok {
			return NewError("'%s' object is not iterable", pyTypeName(args[1]))
		}
		list.Elements = append(list.Elements, other.Elements...)
		return NULL
	}},
	"clear": {Fn: func(args ...Object) Object {
		list := args[0].(*Array)
		list.Elements = list.Elements[:0]
		return NULL
	}},
	"copy": {Fn: func(args ...Object) Object {
		list := args[0].(*Array)
		return &Array{Elements: append([]Object{}, list.Elements...)}
	}},
	"index": {Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return NewError("index() takes exactly one argument (%d given)", len(args)-1)
		}
		list := args[0].(*Array)
		for i, e := range list.Elements {
			if objectsEqual(e, args[1]) {
				return &Integer{Value: int64(i)}
			}
		}
		return NewErrorWithKind("ValueError", "%s is not in list", args[1].Inspect())
	}},
	"count": {Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return NewError("count() takes exactly one argument (%d given)", len(args)-1)
		}
		list := args[0].(*Array)
		count := 0
		for _, e := range list.Elements {
			if objectsEqual(e, args[1]) {
				count++
			}
		}
		return &Integer{Value: int64(count)}
	}},
	"reverse": {Fn: func(args ...Object) Object {
		list := args[0].(*Array)
		for i, j := 0, len(list.Elements)-1; i < j; i, j = i+1, j-1 {
			list.Elements[i], list.Elements[j] = list.Elements[j], list.Elements[i]
		}
		return NULL
	}},
	"sort": {Fn: func(args ...Object) Object {
		list := args[0].(*Array)
		var cmpErr Object
		sort.SliceStable(list.Elements, func(i, j int) bool {
			if cmpErr != nil {
				return false
			}
			c, err := compareValues(list.Elements[i], list.Elements[j])
			if err != nil {
				cmpErr = err
				return false
			}
			return c < 0
		})
		if cmpErr != nil {
			return cmpErr
		}
		return NULL
	}},
}

var dictMethods = map[string]*Builtin{
	"keys": {Fn: func(args ...Object) Object {
		return &Array{Elements: args[0].(*Dict).Keys()}
	}},
	"values": {Fn: func(args ...Object) Object {
		d := args[0].(*Dict)
		vals := make([]Object, 0, d.Len())
		for _, k := range d.keys {
			vals = append(vals, d.pairs[k].Value)
		}
		return &Array{Elements: vals}
	}},
	"items": {Fn: func(args ...Object) Object {
		d := args[0].(*Dict)
		items := make([]Object, 0, d.Len())
		for _, k := range d.keys {
			p := d.pairs[k]
			items = append(items, &Tuple{Elements: []Object{p.Key, p.Value}})
		}
		return &Array{Elements: items}
	}},
	"get": {Fn: func(args ...Object) Object {
		if len(args) < 2 || len(args) > 3 {
			return NewError("get() takes one or two arguments (%d given)", len(args)-1)
		}
		d := args[0].(*Dict)
		if _, hashable := dictKeyString(args[1]); !hashable {
			return NewError("unhashable type: '%s'", pyTypeName(args[1]))
		}
		if v, ok := d.Get(args[1]); ok {
			return v
		}
		if len(args) == 3 {
			return args[2]
		}
		return NULL
	}},
	"clear": {Fn: func(args ...Object) Object {
		d := args[0].(*Dict)
		d.keys = d.keys[:0]
		d.pairs = make(map[string]DictPair)
		return NULL
	}},
}

// stripChars 实现 str.strip(chars)：无参数去空白，有参数去掉两端出现在
// chars 字符串中的字符。left/right 控制只去一侧。
func stripChars(s, chars string, left, right bool) string {
	if chars == "" {
		if left {
			s = strings.TrimLeftFunc(s, unicode.IsSpace)
		}
		if right {
			s = strings.TrimRightFunc(s, unicode.IsSpace)
		}
		return s
	}
	cutset := chars
	if left {
		s = strings.TrimLeft(s, cutset)
	}
	if right {
		s = strings.TrimRight(s, cutset)
	}
	return s
}

// pyCapitalize 首字母大写、其余小写（按 rune，支持中文/全角字符）。
func pyCapitalize(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	out := make([]rune, len(r))
	out[0] = unicode.ToUpper(r[0])
	for i := 1; i < len(r); i++ {
		out[i] = unicode.ToLower(r[i])
	}
	return string(out)
}

var strMethods = map[string]*Builtin{
	"upper": {Fn: func(args ...Object) Object {
		return &String{Value: strings.ToUpper(args[0].(*String).Value)}
	}},
	"lower": {Fn: func(args ...Object) Object {
		return &String{Value: strings.ToLower(args[0].(*String).Value)}
	}},
	"capitalize": {Fn: func(args ...Object) Object {
		return &String{Value: pyCapitalize(args[0].(*String).Value)}
	}},
	"strip": {Fn: func(args ...Object) Object {
		if len(args) > 2 {
			return NewError("strip() takes at most one argument (%d given)", len(args)-1)
		}
		chars := ""
		if len(args) == 2 {
			c, ok := args[1].(*String)
			if !ok {
				return NewError("strip() argument must be str, not %s", pyTypeName(args[1]))
			}
			chars = c.Value
		}
		return &String{Value: stripChars(args[0].(*String).Value, chars, true, true)}
	}},
	"lstrip": {Fn: func(args ...Object) Object {
		chars := ""
		if len(args) == 2 {
			chars = args[1].(*String).Value
		}
		return &String{Value: stripChars(args[0].(*String).Value, chars, true, false)}
	}},
	"rstrip": {Fn: func(args ...Object) Object {
		chars := ""
		if len(args) == 2 {
			chars = args[1].(*String).Value
		}
		return &String{Value: stripChars(args[0].(*String).Value, chars, false, true)}
	}},
	"replace": {Fn: func(args ...Object) Object {
		if len(args) != 3 {
			return NewError("replace() takes exactly two arguments (%d given)", len(args)-1)
		}
		old, ok1 := args[1].(*String)
		new, ok2 := args[2].(*String)
		if !ok1 || !ok2 {
			return NewError("replace() arguments must be str")
		}
		if old.Value == "" {
			return NewError("empty separator")
		}
		return &String{Value: strings.ReplaceAll(args[0].(*String).Value, old.Value, new.Value)}
	}},
	"split": {Fn: func(args ...Object) Object {
		if len(args) > 2 {
			return NewError("split() takes at most one argument (%d given)", len(args)-1)
		}
		s := args[0].(*String).Value
		var parts []string
		if len(args) == 1 {
			parts = strings.Fields(s) // 连续空白切分,无空串
		} else {
			sep, ok := args[1].(*String)
			if !ok {
				return NewError("split() separator must be str, not %s", pyTypeName(args[1]))
			}
			if sep.Value == "" {
				return NewError("ValueError: empty separator")
			}
			parts = strings.Split(s, sep.Value)
		}
		elems := make([]Object, 0, len(parts))
		for _, p := range parts {
			elems = append(elems, &String{Value: p})
		}
		return &Array{Elements: elems}
	}},
	"join": {Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return NewError("join() takes exactly one argument (%d given)", len(args)-1)
		}
		sep := args[0].(*String).Value
		iter, ok := args[1].(*Array)
		if !ok {
			return NewError("can only join an iterable, got %s", pyTypeName(args[1]))
		}
		parts := make([]string, 0, len(iter.Elements))
		for i, e := range iter.Elements {
			s, ok := e.(*String)
			if !ok {
				return NewError("sequence item %d: expected str instance, %s found",
					i, pyTypeName(e))
			}
			parts = append(parts, s.Value)
		}
		return &String{Value: strings.Join(parts, sep)}
	}},
	"startswith": {Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return NewError("startswith() takes exactly one argument (%d given)", len(args)-1)
		}
		prefix, ok := args[1].(*String)
		if !ok {
			return NewError("startswith() argument must be str, not %s", pyTypeName(args[1]))
		}
		return NativeBoolToBooleanObject(strings.HasPrefix(args[0].(*String).Value, prefix.Value))
	}},
	"endswith": {Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return NewError("endswith() takes exactly one argument (%d given)", len(args)-1)
		}
		suffix, ok := args[1].(*String)
		if !ok {
			return NewError("endswith() argument must be str, not %s", pyTypeName(args[1]))
		}
		return NativeBoolToBooleanObject(strings.HasSuffix(args[0].(*String).Value, suffix.Value))
	}},
	"find": {Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return NewError("find() takes exactly one argument (%d given)", len(args)-1)
		}
		sub, ok := args[1].(*String)
		if !ok {
			return NewError("find() argument must be str, not %s", pyTypeName(args[1]))
		}
		return &Integer{Value: int64(strings.Index(args[0].(*String).Value, sub.Value))}
	}},
	"count": {Fn: func(args ...Object) Object {
		if len(args) != 2 {
			return NewError("count() takes exactly one argument (%d given)", len(args)-1)
		}
		sub, ok := args[1].(*String)
		if !ok {
			return NewError("count() argument must be str, not %s", pyTypeName(args[1]))
		}
		return &Integer{Value: int64(strings.Count(args[0].(*String).Value, sub.Value))}
	}},
	"isdigit": {Fn: func(args ...Object) Object {
		s := args[0].(*String).Value
		if s == "" {
			return FALSE_OBJ
		}
		for _, r := range s {
			if !unicode.IsDigit(r) {
				return FALSE_OBJ
			}
		}
		return TRUE_OBJ
	}},
}

func evalIndexExpression(left, index Object) Object {
	switch l := left.(type) {
	case *Array:
		idx, ok := index.(*Integer)
		if !ok {
			return NewError("list indices must be integers, not %s", index.Type())
		}
		i := idx.Value
		if i < 0 {
			i = int64(len(l.Elements)) + i
		}
		if i < 0 || i >= int64(len(l.Elements)) {
			return NewErrorWithKind("IndexError", "list index out of range")
		}
		return l.Elements[i]
	case *Dict:
		v, ok := l.Get(index)
		if !ok {
			return NewErrorWithKind("KeyError", "%s", index.Inspect())
		}
		return v
	case *String:
		idx, ok := index.(*Integer)
		if !ok {
			return NewError("string indices must be integers, not %s", index.Type())
		}
		runes := []rune(l.Value)
		i := idx.Value
		if i < 0 {
			i = int64(len(runes)) + i
		}
		if i < 0 || i >= int64(len(runes)) {
			return NewErrorWithKind("IndexError", "string index out of range")
		}
		return &String{Value: string(runes[i])}
	case *Tuple:
		idx, ok := index.(*Integer)
		if !ok {
			return NewError("tuple indices must be integers, not %s", index.Type())
		}
		i := idx.Value
		if i < 0 {
			i = int64(len(l.Elements)) + i
		}
		if i < 0 || i >= int64(len(l.Elements)) {
			return NewErrorWithKind("IndexError", "tuple index out of range")
		}
		return l.Elements[i]
	}
	return NewError("index operator not supported: %s", left.Type())
}

// evalSliceExpression 实现 seq[start:stop:step]，支持 list/tuple/str。
// start/stop/step 为 nil 表示省略；下标为负或越界时按 CPython 语义截断。
func evalSliceExpression(left, start, stop, step Object) Object {
	switch c := left.(type) {
	case *Array:
		items, err := pySliceItems(c.Elements, start, stop, step)
		if err != nil {
			return err
		}
		return &Array{Elements: items}
	case *Tuple:
		items, err := pySliceItems(c.Elements, start, stop, step)
		if err != nil {
			return err
		}
		return &Tuple{Elements: items}
	case *String:
		runes := []rune(c.Value)
		objs := make([]Object, len(runes))
		for i, r := range runes {
			objs[i] = &String{Value: string(r)}
		}
		items, err := pySliceItems(objs, start, stop, step)
		if err != nil {
			return err
		}
		out := ""
		for _, o := range items {
			out += o.(*String).Value
		}
		return &String{Value: out}
	}
	return NewError("'%s' object is not subscriptable", pyTypeName(left))
}

// pySliceItems 对元素序列应用 CPython 切片规则,返回新元素切片;
// 第二个返回值非 nil 表示切片参数错误。
func pySliceItems(elems []Object, start, stop, step Object) ([]Object, Object) {
	n := int64(len(elems))
	stepVal := int64(1)
	if step != nil {
		s, ok := step.(*Integer)
		if !ok {
			return nil, NewError("slice indices must be integers or None, not %s", step.Type())
		}
		stepVal = s.Value
		if stepVal == 0 {
			return nil, NewError("ValueError: slice step cannot be zero")
		}
	}
	for _, v := range []Object{start, stop} {
		if v != nil {
			if _, ok := v.(*Integer); !ok {
				return nil, NewError("slice indices must be integers or None, not %s", v.Type())
			}
		}
	}
	// asIndex 把省略(nil)/负数/越界的下标归一化成合法区间端点
	asIndex := func(v Object, def int64) int64 {
		if v == nil {
			return def
		}
		val := v.(*Integer).Value
		if val < 0 {
			val += n
		}
		if stepVal > 0 {
			if val < 0 {
				val = 0
			}
			if val > n {
				val = n
			}
		} else {
			if val < -1 {
				val = -1
			}
			if val > n-1 {
				val = n - 1
			}
		}
		return val
	}
	out := []Object{}
	if stepVal > 0 {
		lo := asIndex(start, 0)
		hi := asIndex(stop, n)
		for i := lo; i < hi; i += stepVal {
			out = append(out, elems[i])
		}
		return out, nil
	}
	// 负步长:从高往低走,hi 是开区间下界
	lo := asIndex(start, n-1)
	hi := asIndex(stop, -1)
	for i := lo; i > hi; i += stepVal {
		out = append(out, elems[i])
	}
	return out, nil
}

func evalIndexAssign(target, index, value Object) Object {
	switch t := target.(type) {
	case *Array:
		idxObj, ok := index.(*Integer)
		if !ok {
			return NewError("list index must be integer, not %s", index.Type())
		}
		idx := idxObj.Value
		if idx < 0 {
			idx = int64(len(t.Elements)) + idx
		}
		if idx < 0 || idx >= int64(len(t.Elements)) {
			return NewError("list assignment index out of range")
		}
		t.Elements[idx] = value
		return value
	case *Dict:
		if !t.Set(index, value) {
			return NewError("unhashable type: '%s' as dict key", pyTypeName(index))
		}
		return value
	case *Tuple:
		return NewError("TypeError: 'tuple' object does not support item assignment")
	}
	return NewError("cannot assign to index of %s", target.Type())
}

// ----------------------------------------------------------------------------
// 内建函数
// ----------------------------------------------------------------------------

// builtins 是全局内建名 -> 对象(函数或异常类)。
var builtins = map[string]Object{
	"print": &Builtin{
		Fn: func(args ...Object) Object {
			parts := make([]string, 0, len(args))
			for _, a := range args {
				parts = append(parts, a.Inspect())
			}
			out := joinStrings(parts, " ")
			fmt.Println(out)
			return NULL
		},
	},
	"len": &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return NewError("len() takes exactly 1 argument (%d given)", len(args))
			}
			switch v := args[0].(type) {
			case *String:
				return &Integer{Value: int64(len(v.Value))}
			case *Array:
				return &Integer{Value: int64(len(v.Elements))}
			case *Dict:
				return &Integer{Value: int64(v.Len())}
			case *Tuple:
				return &Integer{Value: int64(len(v.Elements))}
			}
			return NewError("object of type %s has no len()", args[0].Type())
		},
	},
	"range": &Builtin{
		Fn: func(args ...Object) Object {
			// range(stop) 或 range(start, stop) 或 range(start, stop, step)
			if len(args) < 1 || len(args) > 3 {
				return NewError("range() takes 1 to 3 arguments (%d given)", len(args))
			}
			intArgs := make([]int64, len(args))
			for i, a := range args {
				n, ok := a.(*Integer)
				if !ok {
					return NewError("range() integer end argument expected, got %s", a.Type())
				}
				intArgs[i] = n.Value
			}
			var start, stop, step int64 = 0, intArgs[0], 1
			if len(args) >= 2 {
				start = intArgs[0]
				stop = intArgs[1]
			}
			if len(args) == 3 {
				step = intArgs[2]
				if step == 0 {
					return NewError("range() step argument must not be zero")
				}
			}
			elems := []Object{}
			if step > 0 {
				for i := start; i < stop; i += step {
					elems = append(elems, &Integer{Value: i})
				}
			} else {
				for i := start; i > stop; i += step {
					elems = append(elems, &Integer{Value: i})
				}
			}
			return &Array{Elements: elems}
		},
	},
	"str": &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return NewError("str() takes exactly 1 argument (%d given)", len(args))
			}
			return &String{Value: args[0].Inspect()}
		},
	},
	"int": &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return NewError("int() takes exactly 1 argument (%d given)", len(args))
			}
			switch v := args[0].(type) {
			case *Integer:
				return v
			case *Float:
				return &Integer{Value: int64(v.Value)}
			case *String:
				n, err := parseIntFromString(v.Value)
				if err != nil {
					return NewError("invalid literal for int(): %q", v.Value)
				}
				return &Integer{Value: n}
			}
			return NewError("int() argument must be number or string, not %s", args[0].Type())
		},
	},
	"float": &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return NewError("float() takes exactly 1 argument (%d given)", len(args))
			}
			switch v := args[0].(type) {
			case *Float:
				return v
			case *Integer:
				return &Float{Value: float64(v.Value)}
			case *String:
				f, err := parseFloatFromString(v.Value)
				if err != nil {
					return NewError("could not convert string to float: %q", v.Value)
				}
				return &Float{Value: f}
			}
			return NewError("float() argument must be number or string, not %s", args[0].Type())
		},
	},
	"type": &Builtin{
		Fn: func(args ...Object) Object {
			if len(args) != 1 {
				return NewError("type() takes exactly 1 argument (%d given)", len(args))
			}
			return &String{Value: pyTypeName(args[0])}
		},
	},
	"input": &Builtin{
		Fn: func(args ...Object) Object {
			// 简易 input：可选 prompt 参数（无内置 ReadLine，调用方负责注入）
			return NULL
		},
	},
}

func init() {
	// 内建异常类型:可调用生成异常实例(raise ValueError("msg"))
	for _, name := range []string{
		"Exception", "ValueError", "TypeError", "KeyError",
		"IndexError", "NameError", "ZeroDivisionError", "RuntimeError",
	} {
		builtins[name] = &ExceptionClass{Name: name}
	}
}

// pyTypeName 把内部类型名映射成 Python 风格的字符串。
func pyTypeName(o Object) string {
	switch ov := o.(type) {
	case *Integer:
		return "int"
	case *Float:
		return "float"
	case *String:
		return "str"
	case *Boolean:
		return "bool"
	case *Null:
		return "NoneType"
	case *Array:
		return "list"
	case *Dict:
		return "dict"
	case *Tuple:
		return "tuple"
	case *Function:
		return "function"
	case *Builtin:
		return "builtin_function_or_method"
	case *ExceptionClass:
		return "type"
	case *ExceptionInstance:
		return ov.KindName
	}
	return string(o.Type())
}

// parseIntFromString / parseFloatFromString：在不引入 strconv 的前提下做基础转换。
func parseIntFromString(s string) (int64, error) {
	var n int64
	var sign int64 = 1
	i := 0
	if len(s) == 0 {
		return 0, fmt.Errorf("empty")
	}
	if s[0] == '-' {
		sign = -1
		i++
	} else if s[0] == '+' {
		i++
	}
	if i >= len(s) {
		return 0, fmt.Errorf("no digits")
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("non-digit")
		}
		n = n*10 + int64(c-'0')
	}
	return n * sign, nil
}

func parseFloatFromString(s string) (float64, error) {
	var n float64
	var sign float64 = 1
	i := 0
	if len(s) == 0 {
		return 0, fmt.Errorf("empty")
	}
	if s[0] == '-' {
		sign = -1
		i++
	} else if s[0] == '+' {
		i++
	}
	for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		n = n*10 + float64(s[i]-'0')
	}
	if i < len(s) && s[i] == '.' {
		i++
		var frac float64 = 0.1
		for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
			n += float64(s[i]-'0') * frac
			frac *= 0.1
		}
	}
	if i < len(s) {
		return 0, fmt.Errorf("trailing chars")
	}
	return n * sign, nil
}