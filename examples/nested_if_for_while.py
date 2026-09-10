# nested_if_for_while.py - 综合嵌套：函数/if/while/for 互相嵌套

def classify(x):
    if x < 0:
        return "negative"
    elif x == 0:
        return "zero"
    else:
        if x < 10:
            return "small"
        else:
            return "big"

for i in [-3, 0, 5, 20]:
    print(i, "->", classify(i))

# while 嵌套 if-else
n = 10
while n > 0:
    if n % 2 == 0:
        print("even", n)
    else:
        print("odd ", n)
    n = n - 1

# for 嵌套 if-elif-elif-else
for x in [1, 2, 3, 4, 5, 6]:
    if x == 1:
        print("one")
    elif x == 2:
        print("two")
    elif x < 5:
        print("3-4", x)
    else:
        print(">=5", x)

# 双重 for 嵌套
for i in range(3):
    for j in range(3):
        print(i, j)