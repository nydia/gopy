# factorial.py - 递归 + while 循环两种写法

def fact_rec(n):
    if n <= 1:
        return 1
    return n * fact_rec(n - 1)

def fact_iter(n):
    r = 1
    i = 1
    while i <= n:
        r = r * i
        i = i + 1
    return r

print("fact_rec(6) =", fact_rec(6))
print("fact_iter(6) =", fact_iter(6))