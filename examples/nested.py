# nested.py - 嵌套函数 / 闭包

def make_counter():
    n = 0
    def inc():
        n = n + 1
        return n
    return inc

c = make_counter()
print(c())
print(c())
print(c())