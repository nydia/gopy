# -*- coding: utf-8 -*-
"""R3: 运算符增强。

- 三元表达式 `a if c else b`(修复:IF 缺优先级导致不可达的潜伏 bug)
- `in` / `not in` 成员运算(str / list / dict)
- 字符串比较 == != < > <= >= 与重复 `s * n` / `n * s`
- 跨类型相等:1 == "1" 应为 False 而不是报错(对齐 CPython)
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from harness import expect_output, expect_error, run_tests


def test_ternary_basic():
    expect_output(
        "x = 1 if True else 2\nprint(x)\ny = 1 if False else 2\nprint(y)\n",
        ["1", "2"],
        name="ternary-basic")


def test_ternary_with_condition():
    expect_output(
        'n = 7\nprint("even" if n % 2 == 0 else "odd")\nprint("even" if 0 else "odd")\n',
        ["odd", "odd"],
        name="ternary-condition")


def test_ternary_nested_and_in_call():
    expect_output(
        "a = 1\nb = 2\nprint(a if a > b else b)\n"
        "x = \"big\" if a > b else (\"small\" if a < b else \"eq\")\nprint(x)\n",
        ["2", "small"],
        name="ternary-nested")


def test_in_string():
    expect_output(
        'print("ell" in "hello")\nprint("z" in "hello")\nprint("z" not in "hello")\n',
        ["True", "False", "True"],
        name="in-string")


def test_in_list():
    expect_output(
        "print(2 in [1, 2, 3])\nprint(5 in [1, 2, 3])\nprint(5 not in [1, 2, 3])\n"
        'print("a" in ["a", "b"])\n',
        ["True", "False", "True", "True"],
        name="in-list")


def test_in_dict():
    expect_output(
        'd = {"a": 1, 2: "x"}\nprint("a" in d)\nprint("b" in d)\nprint(2 in d)\nprint("b" not in d)\n',
        ["True", "False", "True", "True"],
        name="in-dict")


def test_in_precedence_with_not():
    # not in 与 and/or 组合
    expect_output(
        'word = "hello"\nif "ell" in word and "x" not in word:\n    print("both")\n',
        "both",
        name="in-with-and")


def test_string_compare():
    expect_output(
        'print("abc" == "abc")\nprint("abc" != "abd")\nprint("a" < "b")\n'
        'print("apple" < "banana")\nprint("b" > "a")\nprint("a" >= "a")\n',
        ["True", "True", "True", "True", "True", "True"],
        name="string-compare")


def test_string_repeat():
    expect_output(
        'print("ab" * 3)\nprint(3 * "ab")\nprint("x" * 0)\nprint("-" * 5)\n',
        ["ababab", "ababab", "", "-----"],
        name="string-repeat")


def test_cross_type_equality():
    # CPython:不同类型 == 返回 False 而不是报错
    expect_output(
        'print(1 == "1")\nprint(1 != "1")\nprint([1, 2] == [1, 2])\nprint([1] == "a")\n',
        ["False", "True", "True", "False"],
        name="cross-type-equality")


def test_int_float_equality():
    expect_output(
        "print(1 == 1.0)\nprint(2.5 == 2.5)\n",
        ["True", "True"],
        name="int-float-equality")


def test_in_error_cases():
    expect_error('print(1 in "abc")\n', "in <string>", name="in-string-nonstr-left")
    expect_error('print(1 in 123)\n', "not iterable", name="in-non-iterable")


def main():
    tests = [
        ("三元表达式(回归)", test_ternary_basic),
        ("三元条件求值", test_ternary_with_condition),
        ("三元嵌套与调用参数内", test_ternary_nested_and_in_call),
        ("in / not in 字符串", test_in_string),
        ("in / not in 列表", test_in_list),
        ("in / not in 字典(键)", test_in_dict),
        ("成员运算与 and/or 组合", test_in_precedence_with_not),
        ("字符串比较", test_string_compare),
        ("字符串重复", test_string_repeat),
        ("跨类型相等语义", test_cross_type_equality),
        ("int 与 float 相等", test_int_float_equality),
        ("成员运算错误路径", test_in_error_cases),
    ]
    ok = run_tests(tests)
    if not ok:
        sys.exit(1)
    print("PASS test_03_operators")


if __name__ == "__main__":
    main()
