# fib.py - 斐波那契：while 循环 + 函数 + 递归

def fib(n):
    if n < 2:
        return n
    return fib(n - 1) + fib(n - 2)

i = 0
while i < 10:
    print("fib(", i, ") =", fib(i))
    i = i + 1