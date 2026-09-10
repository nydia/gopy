# gopy
gopy 是一个教学导向的 Python 子集解释器，整体结构参考 Writing an Interpreter in Go，但面向真实 Python 语法而非 Monkey：缩进敏感的 lexer、显式的 INDENT/DEDENT token、for-in 遍历、elif、列表/字典/元组、切片、异常传播、闭包等。 仅依赖 Go 标准库，约 4 800 行 Go 代码，0 第三方依赖。
