# hello.py - 基础字面量与算术

print("Hello, gopy!")

x = 1 + 2 * 3
print("x =", x)

y = x * (x + 1) / 2
print("y =", y)

name = "gopy"
greeting = "hello, " + name
print(greeting)

flag = True and not False
print("flag =", flag)

n = -5
abs_n = 0
if n >= 0:
    abs_n = n
else:
    abs_n = -n
print("abs(-5) =", abs_n)