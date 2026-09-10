package main

// Environment 是一个作用域，记录变量名到对象的绑定。
// 它持有一个 outer（外层作用域），实现嵌套作用域 / 闭包。
type Environment struct {
	store  map[string]Object
	outer   *Environment
	constants map[string]bool // 标记常量（如 def 定义的函数），禁止重新赋值
}

// NewEnvironment 构造一个没有外层作用域的环境（最顶层）。
func NewEnvironment() *Environment {
	return &Environment{
		store:    make(map[string]Object),
		outer:    nil,
		constants: make(map[string]bool),
	}
}

// NewEnclosedEnvironment 构造一个内嵌在外层环境中的子环境。
// 用于：函数调用、块语句、模块。
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get 查找一个名字。从当前作用域开始，沿 outer 链向上。
func (e *Environment) Get(name string) (Object, bool) {
	val, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return val, ok
}

// Set 在当前作用域中绑定一个名字。
// 但如果当前作用域有 outer，且 outer 已经存在同名变量，则写到 outer（实现 closure）。
// 这样嵌套函数内的赋值能修改外层函数的局部变量。
func (e *Environment) Set(name string, val Object) Object {
	if e.outer != nil {
		if _, ok := e.outer.store[name]; ok {
			e.outer.Set(name, val)
			return val
		}
	}
	e.store[name] = val
	return val
}

// SetConst 在当前作用域中绑定一个常量（不允许后续 Set 覆盖）。
// def 的函数名走这条路径。
func (e *Environment) SetConst(name string, val Object) Object {
	e.constants[name] = true
	e.store[name] = val
	return val
}

// IsConst 询问某个名字在当前作用域是不是常量（用于 def 函数名重复定义的检查）。
func (e *Environment) IsConst(name string) bool {
	if e == nil {
		return false
	}
	if c, ok := e.constants[name]; ok && c {
		return true
	}
	if e.outer != nil {
		return e.outer.IsConst(name)
	}
	return false
}