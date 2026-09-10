# gopy

[![Go Version](https://img.shields.io/badge/Go-1.24.1-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)
[![Dependencies](https://img.shields.io/badge/dependencies-0-success)](#)
[![Standard Library Only](https://img.shields.io/badge/std--lib-only-blueviolet)](#)

> 用 Go 写一个 Python 子集解释器 —— 树遍历式（tree-walking）实现：Lexer → Pratt Parser → Tree-walking Evaluator。

`gopy` 是一个教学导向的 Python 子集解释器，整体结构参考 [*Writing an Interpreter in Go*](https://interpreterbook.com/)，但面向真实 Python 语法而非 Monkey：缩进敏感的 lexer、显式的 `INDENT`/`DEDENT` token、`for-in` 遍历、`elif`、列表/字典/元组、切片、异常传播、闭包等。

仅依赖 Go 标准库，约 4 800 行 Go 代码，0 第三方依赖。

---

## 特性一览

| 类别 | 实现 |
| --- | --- |
| **字面量** | `int` · `float` · `str` · `True` / `False` · `None` · 列表 `[]` · 字典 `{...}` · 元组 `(a,)` |
| **运算符** | `+ - * / // %` · 比较 `== != < > <= >=` · 成员 `in / not in` · 布尔 `and / or / not`（短路）· 一元 `-` · 字符串/列表 `*` 重复 · `+` 拼接 |
| **控制流** | `if / elif / else` · `while` · `for x in iter` · `for k, v in dict.items` · `break / continue / pass` · 三元 `a if c else b` |
| **函数** | `def` / `return` · 位置参数 · 关键字参数 · 默认值 · 闭包（`Environment.Set` 写穿外层） |
| **数据结构** | 列表下标/切片/赋值 · 字典插入序保持 · 元组不可变 · 序列解包 `a, b = x` · 复合赋值 `+= -= *= /= //=` |
| **方法调用** | `obj.method(...)` 语法 · list/dict/str 全套方法 · 方法是一等值可赋给变量 |
| **异常** | `try / except [Kind as name] / raise` · 内建异常类（`ValueError` 等）· 异常按 `Kind` 匹配，跨函数/跨循环传播 |
| **可迭代** | list · tuple · dict(键) · str(逐 rune) · `range` |
| **内建** | `print` · `len` · `range` · `str` · `int` · `float` · `type` · `input`(stub) |
| **兼容性** | 行尾 `\n` / `\r\n` / `\r` 全部接受 · 数值语义对齐 CPython（`/` 真除 / `//` 向下取整 / `%` 符号随除数）|

---

## 快速开始

```bash
# 1) 克隆与构建
git clone https://github.com/lvhuaqiang/gopy.git
cd gopy
go build -o gopy .

# 2) 跑个例子
./gopy examples/hello.py

# 3) 或进入 REPL
./gopy
>>> print("hi", 1+1)
hi 2
```

### 调试开关

```bash
GOPY_TOKENS=1 ./gopy examples/list_ops.py   # dump token 流（缩进 INDENT/DEDENT 也可见）
GOPY_AST=1    ./gopy examples/list_ops.py   # dump AST
```

### 测试

测试本身是 Python 脚本（`test/test_NN_*.py`），把内嵌源码灌给 `gopy` 解释器并断言 stdout/退出码。

```bash
python3 test/run_all.py
```

---

## 代码示例

```python
# examples/hello.py
print("Hello, gopy!")

x = 1 + 2 * 3
y = x * (x + 1) / 2   # / 永远是真除：14.0

# 字典保持插入序，in 检查 key
d = {"a": 1, "b": 2}
for k, v in d.items():
    print(k, v)

# 切片 [a:b:c] 负数/步长都支持
s = "Hello, 世界"
print(s[7:])              # 世界
print(s[::-1])            # 界世 ,olleH

# 异常
try:
    int("abc")
except ValueError as e:
    print("bad:", e)
```

```python
# 闭包：make_counter
def make_counter():
    n = 0
    def step():
        n = n + 1   # 写穿外层（Environment.Set）
        return n
    return step

c = make_counter()
print(c(), c(), c())  # 1 2 3
```

更多见 [`examples/`](./examples)。

---

## 架构

```
gopy/
├── main.go              入口：脚本 / REPL / token+AST dump
├── repl.go              REPL 与文件执行
├── token.go             TokenType + 字面量映射
├── lexer.go             缩进敏感的 lexer（INDENT/DEDENT/CRLF）
├── ast.go               全部 AST 节点
├── parser.go            Pratt 解析器
├── object.go            运行时对象（Integer / Float / String / Array / Dict / Tuple / Function / BoundMethod / Error / ExceptionInstance ...）
├── environment.go       作用域链 + 闭包写穿
├── evaluator.go         AST 求值 + 内建函数 + 内建异常类
├── examples/            示例 Python 脚本
└── test/                黑盒测试（每轮一个 test_NN_*.py）
    ├── harness.py
    └── run_all.py
```

### 三段式管线

```
源码 ─▶ Lexer ─▶ Token 流 ─▶ Pratt Parser ─▶ AST ─▶ Tree-walking Evaluator ─▶ Object
```

- **Lexer**：状态化的缩进栈，进入/退出块时比较当前缩进与栈顶，emit `INDENT` / `DEDENT`；支持 `\n` / `\r\n` / `\r` 三种行尾。
- **Pratt Parser**：用 prefix / infix 函数表 + 优先级表解析表达式；通过 `noIn` 标志处理 `for x in iter` 中 `in` 与成员运算符 `in` 的歧义；用 `peekTokenIs(ELIF/ELSE)` 解决 `DEDENT` 后紧随的分支判断。
- **Evaluator**：纯 tree-walking，每条语句/表达式递归求值；`Environment` 形成作用域链，`Set` 写穿到定义层（闭包）；`and`/`or` 在通用中缀前拦截实现短路；`try/except` 沿调用栈冒泡。

### 块语句退出约定

`parseBlockStatement` 结束时 `curToken = DEDENT`，由调用方消费。`while` / `for` / `def` 不在末尾 `nextToken`，靠 `ParseProgram` 的空白 token 跳过循环统一处理。`if/elif/else` 用 `peekTokenIs(ELIF/ELSE)` 而非 `curTokenIs` 判断下一段——因为 consequence block 结束后 `cur` 停在 `DEDENT` 上。

---

## 已知限制

- 集合、生成器、推导式
- 类与 OOP（`obj.attr` 属性访问与 `obj.m()` 方法调用已就绪，缺 `class` / 实例）
- `try/except/finally` 的 `finally` 子句
- 用户自定义异常类（只能 `raise` 内建异常类）
- 异常继承体系（`except` 按异常名精确匹配，无父子类）
- f-string / `str.format`
- 多行表达式（`\` 续行）
- `import` / 模块系统
- 文件 IO（`open()` 等）
- Unicode 标识符（按字节切分，中文等非 ASCII 字符目前仅在字符串字面量与下标中按 rune 处理）

---

## Roadmap

- [ ] 异常继承（`except` 支持父子类）
- [ ] `finally` 子句
- [ ] 集合 `set` / 推导式
- [ ] 类与 `class` 语句
- [ ] `import` / 模块加载
- [ ] 字节码编译（Monkey 第 II 部分风格的字节码 VM）

---

## 致谢

- 整体结构深受 [*Writing an Interpreter in Go*](https://interpreterbook.com/)（Thorsten Ball）影响。
- 数值语义、切片归一化、字典键规范化、容器 `repr` 打印等细节对齐 [CPython](https://github.com/python/cpython) 行为。
- 标点 / emoji 配色参考 GitHub README 通用约定。

---

## License

[MIT](./LICENSE) — Copyright (c) 2026 lvhuaqiang.
