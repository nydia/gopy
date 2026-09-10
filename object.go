package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Object 是运行时所有值的统一接口。
// 用 Type() + Inspect() 让 evaluator 能多态地处理不同类型的对象。
type Object interface {
	Type() ObjectType
	Inspect() string
}

// ObjectType 标识运行时对象的种类。
type ObjectType string

const (
	INTEGER_OBJ     = "INTEGER"
	FLOAT_OBJ       = "FLOAT"
	STRING_OBJ      = "STRING"
	BOOLEAN_OBJ     = "BOOLEAN"
	NULL_OBJ        = "NULL"
	RETURN_VALUE_OBJ = "RETURN_VALUE"
	FUNCTION_OBJ    = "FUNCTION"
	ARRAY_OBJ       = "ARRAY"
	DICT_OBJ        = "DICT"
	TUPLE_OBJ       = "TUPLE"
	EXCEPTION_CLASS_OBJ = "EXCEPTION_CLASS"
	EXCEPTION_OBJ   = "EXCEPTION"
	BUILTIN_OBJ     = "BUILTIN"
	ERROR_OBJ       = "ERROR"
	// 控制流信号（用于 break/continue）
	BREAK_OBJ    = "BREAK"
	CONTINUE_OBJ = "CONTINUE"
)

// ----------------------------------------------------------------------------
// 基础对象
// ----------------------------------------------------------------------------

type Integer struct{ Value int64 }

func (i *Integer) Type() ObjectType  { return INTEGER_OBJ }
func (i *Integer) Inspect() string   { return fmt.Sprintf("%d", i.Value) }

type Float struct{ Value float64 }

func (f *Float) Type() ObjectType { return FLOAT_OBJ }

// Inspect 对齐 CPython 的 float repr：整值带 ".0"（5.0），其余用最短可回读形式。
func (f *Float) Inspect() string {
	s := strconv.FormatFloat(f.Value, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

type String struct{ Value string }

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type Boolean struct{ Value bool }

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }

// Inspect 用 Python 风格打印布尔值（True/False 而非 true/false）。
func (b *Boolean) Inspect() string {
	if b.Value {
		return "True"
	}
	return "False"
}

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "None" }

// ----------------------------------------------------------------------------
// 函数
// ----------------------------------------------------------------------------

// Function 是用户自定义函数对象。
// 闭包：函数定义处可见的 Environment 会被捕获。
// Defaults 与 Parameters 对齐，保存 def 时求值一次的默认值
//（CPython 语义:可变默认值跨调用共享），nil 表示无默认值。
type Function struct {
	Name       string
	Parameters []*Parameter
	Defaults   []Object
	Body       *BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}
	return fmt.Sprintf("def %s(%s) {...}", f.Name, joinStrings(params, ", "))
}

// ----------------------------------------------------------------------------
// 列表
// ----------------------------------------------------------------------------

type Array struct{ Elements []Object }

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	elems := []string{}
	for _, e := range a.Elements {
		elems = append(elems, reprObject(e))
	}
	return fmt.Sprintf("[%s]", joinStrings(elems, ", "))
}

// ----------------------------------------------------------------------------
// 字典（保持插入序）
// ----------------------------------------------------------------------------

// DictPair 保存字典的一个键值对，Key 保留原始键对象用于打印与遍历。
type DictPair struct {
	Key   Object
	Value Object
}

// Dict 是 Python 风格的字典。Go map 本身无序，所以用 keys 切片记录
// 规范化键的插入顺序，保证打印与 for-in 遍历按插入序进行。
// 键支持 String / Integer / Boolean（bool 与 int 在 Python 中是同一个键）。
type Dict struct {
	keys  []string
	pairs map[string]DictPair
}

func NewDict() *Dict {
	return &Dict{pairs: make(map[string]DictPair)}
}

// dictKeyString 把键对象规范化成内部 map 键；不可哈希的类型返回 false。
// 注意 "1" 与 1 规范化后不同（s:1 与 i:1），与 CPython 语义一致。
func dictKeyString(o Object) (string, bool) {
	switch v := o.(type) {
	case *String:
		return "s:" + v.Value, true
	case *Integer:
		return "i:" + fmt.Sprintf("%d", v.Value), true
	case *Boolean:
		if v.Value {
			return "i:1", true
		}
		return "i:0", true
	}
	return "", false
}

func (d *Dict) Type() ObjectType { return DICT_OBJ }

// Len 返回键值对个数。
func (d *Dict) Len() int { return len(d.keys) }

// Get 按键取值；键不存在或类型不可哈希返回 false。
func (d *Dict) Get(key Object) (Object, bool) {
	k, ok := dictKeyString(key)
	if !ok {
		return nil, false
	}
	p, ok := d.pairs[k]
	return p.Value, ok
}

// Set 插入或更新键值对；新键追加到插入序末尾。
// 键不可哈希时返回 false。
func (d *Dict) Set(key, value Object) bool {
	k, ok := dictKeyString(key)
	if !ok {
		return false
	}
	if _, exists := d.pairs[k]; !exists {
		d.keys = append(d.keys, k)
	}
	d.pairs[k] = DictPair{Key: key, Value: value}
	return true
}

// Keys 按插入序返回键对象列表。
func (d *Dict) Keys() []Object {
	out := make([]Object, 0, len(d.keys))
	for _, k := range d.keys {
		out = append(out, d.pairs[k].Key)
	}
	return out
}

func (d *Dict) Inspect() string {
	parts := []string{}
	for _, k := range d.keys {
		p := d.pairs[k]
		parts = append(parts, reprObject(p.Key)+": "+reprObject(p.Value))
	}
	return "{" + joinStrings(parts, ", ") + "}"
}

// reprObject 返回对象的 Python repr 形式：字符串加单引号，其余同 Inspect。
// CPython 打印容器（list/dict）时元素按 repr 显示，这里与之对齐。
func reprObject(o Object) string {
	if s, ok := o.(*String); ok {
		return "'" + s.Value + "'"
	}
	return o.Inspect()
}

// ----------------------------------------------------------------------------
// 异常类与异常实例
// ----------------------------------------------------------------------------

// ExceptionClass 是异常类型（ValueError 等），可调用得到实例。
type ExceptionClass struct{ Name string }

func (ec *ExceptionClass) Type() ObjectType { return EXCEPTION_CLASS_OBJ }
func (ec *ExceptionClass) Inspect() string  { return "<class '" + ec.Name + "'>" }

// ExceptionInstance 是抛出的异常值：类型名 + 消息。
type ExceptionInstance struct {
	KindName string
	Message  string
}

func (ei *ExceptionInstance) Type() ObjectType { return EXCEPTION_OBJ }
func (ei *ExceptionInstance) Inspect() string {
	if ei.Message != "" {
		return ei.KindName + ": " + ei.Message
	}
	return ei.KindName
}

// ----------------------------------------------------------------------------
// 元组（不可变序列）
// ----------------------------------------------------------------------------

type Tuple struct{ Elements []Object }

func (t *Tuple) Type() ObjectType { return TUPLE_OBJ }

// Inspect 对齐 CPython：单元素元组带尾逗号 (1,)，空元组 ()。
func (t *Tuple) Inspect() string {
	elems := []string{}
	for _, e := range t.Elements {
		elems = append(elems, reprObject(e))
	}
	if len(elems) == 1 {
		return "(" + elems[0] + ",)"
	}
	return "(" + joinStrings(elems, ", ") + ")"
}

// ----------------------------------------------------------------------------
// 内建函数
// ----------------------------------------------------------------------------

// BuiltinFunction 是内建函数的实现签名：接收参数对象数组，返回结果对象。
type BuiltinFunction func(args ...Object) Object

type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

// BoundMethod 是绑定到接收者的方法（如 d.get 中的 get）。
// 调用时接收者自动作为第一个实参传入 Fn。
type BoundMethod struct {
	Receiver Object
	Name     string
	Fn       BuiltinFunction
}

func (bm *BoundMethod) Type() ObjectType { return BUILTIN_OBJ }
func (bm *BoundMethod) Inspect() string {
	return fmt.Sprintf("built-in method %s of %s object", bm.Name, pyTypeName(bm.Receiver))
}

// ----------------------------------------------------------------------------
// 控制流信号
// ----------------------------------------------------------------------------

type ReturnValue struct{ Value Object }

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

type Break struct{}

func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  { return "break" }

type Continue struct{}

func (c *Continue) Type() ObjectType { return CONTINUE_OBJ }
func (c *Continue) Inspect() string  { return "continue" }

// ----------------------------------------------------------------------------
// 错误对象
// ----------------------------------------------------------------------------

// Error 把运行时错误作为值传递，控制流能优雅地冒泡到顶层而不 panic。
// Kind 是异常类型名（如 "ValueError"），空表示未分类（任何类型化 except 都不匹配）。
type Error struct {
	Kind    string
	Message string
	Line    int
	Column  int
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string {
	msg := e.Message
	if e.Kind != "" {
		msg = e.Kind + ": " + msg
	}
	if e.Line > 0 {
		return fmt.Sprintf("Error at %d:%d: %s", e.Line, e.Column, msg)
	}
	return fmt.Sprintf("Error: %s", msg)
}

// NewError 快速构造错误对象（未分类异常）。
func NewError(format string, a ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, a...)}
}

// NewErrorWithKind 构造带类型名的错误对象，供类型化 except 匹配。
func NewErrorWithKind(kind, format string, a ...interface{}) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, a...)}
}

// IsError 用来在 evaluator 中判断一个对象是不是错误——很常用的 helper。
func IsError(obj Object) bool {
	if obj != nil {
		return obj.Type() == ERROR_OBJ
	}
	return false
}

// 标准布尔对象常量，避免重复分配
var (
	TRUE_OBJ  = &Boolean{Value: true}
	FALSE_OBJ = &Boolean{Value: false}
	NULL      = &Null{}
)

// NativeBoolToBooleanObject 把 Go bool 转成 Boolean 对象。
func NativeBoolToBooleanObject(input bool) *Boolean {
	if input {
		return TRUE_OBJ
	}
	return FALSE_OBJ
}