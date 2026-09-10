# -*- coding: utf-8 -*-
"""R9: 切片 [a:b:c]。

- list / str / tuple 切片;负数下标;默认 start/stop;步长(含负步长)
- 越界自动截断(Python 语义);步长为 0 报错
- 字符串按 rune 切片,中文安全
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from harness import expect_output, expect_error, run_tests


def test_list_slice_basic():
    expect_output(
        "nums = [1, 2, 3, 4, 5]\nprint(nums[1:3])\nprint(nums[:2])\nprint(nums[2:])\nprint(nums[:])\n",
        ["[2, 3]", "[1, 2]", "[3, 4, 5]", "[1, 2, 3, 4, 5]"],
        name="list-slice-basic")


def test_list_slice_negative():
    expect_output(
        "nums = [1, 2, 3, 4, 5]\nprint(nums[-2:])\nprint(nums[:-2])\nprint(nums[-3:-1])\n",
        ["[4, 5]", "[1, 2, 3]", "[3, 4]"],
        name="list-slice-negative")


def test_list_slice_step():
    expect_output(
        "nums = [1, 2, 3, 4, 5]\nprint(nums[::2])\nprint(nums[1::2])\nprint(nums[::-1])\nprint(nums[4:1:-1])\n",
        ["[1, 3, 5]", "[2, 4]", "[5, 4, 3, 2, 1]", "[5, 4, 3]"],
        name="list-slice-step")


def test_list_slice_clamp():
    # 切片越界自动截断不报错(普通下标越界仍抛 IndexError,见 test_10)
    expect_output(
        "nums = [1, 2, 3]\nprint(nums[5:])\nprint(nums[-10:2])\nprint(nums[1:100])\n",
        ["[]", "[1, 2]", "[2, 3]"],
        name="list-slice-clamp")


def test_list_index_oob_is_error():
    # R10 起:普通下标越界抛 IndexError(对齐 CPython),可被 try/except 捕获
    from harness import expect_error as _expect_error
    _expect_error(
        "nums = [1, 2, 3]\nprint(nums[5])\n",
        "IndexError",
        name="list-index-oob")


def test_string_slice():
    expect_output(
        'print("hello"[1:4])\nprint("hello"[:3])\nprint("hello"[::-1])\nprint("hello"[-2:])\n',
        ["ell", "hel", "olleh", "lo"],
        name="string-slice")


def test_string_slice_rune_safe():
    expect_output(
        'print("你好世界"[::-1])\nprint("你好世界"[1:3])\n',
        ["界世好你", "好世"],
        name="string-slice-rune")


def test_tuple_slice():
    expect_output(
        't = (1, 2, 3, 4)\nprint(t[1:3])\nprint(t[::-1])\nprint(type(t[1:3]))\n',
        ["(2, 3)", "(4, 3, 2, 1)", "tuple"],
        name="tuple-slice")


def test_slice_step_zero_error():
    expect_error(
        "nums = [1, 2, 3]\nprint(nums[::0])\n",
        "slice step cannot be zero",
        name="slice-step-zero")


def test_slice_of_expression():
    expect_output(
        'print([1, 2, 3, 4][1:-1])\nprint("abcdef".split("c")[0])\n'
        'words = ["alpha", "beta", "gamma"]\nprint(words[:2])\n',
        ["[2, 3]", "ab", "['alpha', 'beta']"],
        name="slice-expression")


def test_slice_in_logic():
    # 切片结果参与运算/比较
    expect_output(
        's = "hello world"\nif s[0:5] == "hello":\n    print("prefix ok")\n'
        "nums = [1, 2, 3, 4]\nprint(nums[:2] + nums[2:])\n",
        ["prefix ok", "[1, 2, 3, 4]"],
        name="slice-in-logic")


def main():
    tests = [
        ("list 切片基础", test_list_slice_basic),
        ("list 负数切片", test_list_slice_negative),
        ("list 步长切片", test_list_slice_step),
        ("list 越界截断", test_list_slice_clamp),
        ("字符串切片", test_string_slice),
        ("中文按 rune 切片", test_string_slice_rune_safe),
        ("元组切片", test_tuple_slice),
        ("步长为 0 报错", test_slice_step_zero_error),
        ("对表达式结果切片", test_slice_of_expression),
        ("切片参与比较与拼接", test_slice_in_logic),
    ]
    ok = run_tests(tests)
    if not ok:
        sys.exit(1)
    print("PASS test_09_slices")


if __name__ == "__main__":
    main()
