# -*- coding: utf-8 -*-
"""R8: 函数默认参数与关键字参数。

- def f(a, b=2, c=3): 位置参数按序,默认参数可省略
- 调用支持关键字参数 f(b=5)、混合 f(1, c=9)
- 错误路径:未知关键字、缺必填参数、过多位置参数、多值冲突、
  非默认参数跟在默认参数后(解析错)、位置参数跟在关键字参数后(解析错)
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from harness import expect_output, expect_error, run_tests


def test_default_args_basic():
    expect_output(
        'def greet(name, greeting="hello"):\n    return greeting + ", " + name\n'
        'print(greet("Alice"))\nprint(greet("Bob", "hi"))\n',
        ["hello, Alice", "hi, Bob"],
        name="default-args-basic")


def test_default_args_multiple():
    expect_output(
        "def f(a, b=2, c=3):\n    return a + b + c\n"
        "print(f(10))\nprint(f(10, 20))\nprint(f(10, 20, 30))\n",
        ["15", "33", "60"],
        name="default-args-multiple")


def test_keyword_args_basic():
    expect_output(
        "def add(a, b):\n    return a - b\n"
        "print(add(a=10, b=3))\nprint(add(b=3, a=10))\nprint(add(10, b=3))\n",
        ["7", "7", "7"],
        name="keyword-args")


def test_keyword_args_override_order():
    expect_output(
        'def info(name, age=0):\n    return name + ":" + str(age)\n'
        'print(info(age=18, name="Li"))\n',
        "Li:18",
        name="keyword-order-free")


def test_default_args_recursion():
    expect_output(
        "def fact(n, acc=1):\n    if n <= 1:\n        return acc\n    return fact(n - 1, acc * n)\n"
        "print(fact(5))\nprint(fact(3, 10))\n",
        ["120", "60"],
        name="default-recursion")


def test_unknown_keyword_error():
    expect_error(
        "def f(a):\n    return a\nf(1, b=2)\n",
        "unexpected keyword argument",
        name="unknown-kwarg")


def test_missing_required_arg_error():
    expect_error(
        "def f(a, b):\n    return a + b\nf(1)\n",
        "missing required argument",
        name="missing-arg")


def test_too_many_positional_error():
    expect_error(
        "def f(a):\n    return a\nf(1, 2)\n",
        "positional argument",
        name="too-many-positional")


def test_multiple_values_error():
    expect_error(
        "def f(a):\n    return a\nf(1, a=2)\n",
        "multiple values for argument",
        name="multiple-values")


def test_non_default_after_default_parse_error():
    expect_error(
        "def f(a=1, b):\n    return a + b\n",
        "non-default argument follows default argument",
        name="non-default-after-default")


def test_positional_after_keyword_parse_error():
    expect_error(
        "def f(a, b):\n    return a + b\nf(a=1, 2)\n",
        "positional argument follows keyword argument",
        name="positional-after-keyword")


def test_default_args_with_lists():
    # CPython 语义:默认值在 def 时求值一次,可变默认值跨调用共享
    expect_output(
        "def push(item, nums=[0]):\n    nums.append(item)\n    return nums\n"
        "print(push(1))\nprint(push(2))\n",
        ["[0, 1]", "[0, 1, 2]"],
        name="default-mutable-shared")


def main():
    tests = [
        ("默认参数基础", test_default_args_basic),
        ("多个默认参数", test_default_args_multiple),
        ("关键字参数", test_keyword_args_basic),
        ("关键字参数顺序自由", test_keyword_args_override_order),
        ("默认参数+递归", test_default_args_recursion),
        ("未知关键字报错", test_unknown_keyword_error),
        ("缺必填参数报错", test_missing_required_arg_error),
        ("过多位置参数报错", test_too_many_positional_error),
        ("多值冲突报错", test_multiple_values_error),
        ("非默认跟默认解析错", test_non_default_after_default_parse_error),
        ("位置跟关键字解析错", test_positional_after_keyword_parse_error),
        ("可变默认值共享语义", test_default_args_with_lists),
    ]
    ok = run_tests(tests)
    if not ok:
        sys.exit(1)
    print("PASS test_08_functions")


if __name__ == "__main__":
    main()
