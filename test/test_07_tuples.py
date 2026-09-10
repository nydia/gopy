# -*- coding: utf-8 -*-
"""R7: 元组与序列解包。

- 元组字面量 (1, 2) / (1,) / (),不可变,支持下标/len/in/for-in/比较
- 解包赋值:a, b = 1, 2(隐式元组)、a, b = [1, 2]、a, b = (1, 2)、a, b = "xy"
- for 双变量解包:for k, v in d.items()
- 元组不可下标赋值
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from harness import expect_output, expect_error, run_tests


def test_tuple_literal_and_print():
    expect_output(
        't = (1, "a")\nprint(t)\nsingle = (1,)\nprint(single)\nempty = ()\nprint(empty)\nprint(len(t))\n',
        ["(1, 'a')", "(1,)", "()", "2"],
        name="tuple-literal")


def test_tuple_index_and_len():
    expect_output(
        't = (10, 20, 30)\nprint(t[0])\nprint(t[-1])\nprint(len(t))\nprint(type(t))\n',
        ["10", "30", "3", "tuple"],
        name="tuple-index")


def test_tuple_index_oob():
    expect_error(
        "t = (1, 2)\nprint(t[5])\n",
        "tuple index out of range",
        name="tuple-index-oob")


def test_tuple_immutable():
    expect_error(
        "t = (1, 2)\nt[0] = 9\n",
        "does not support item assignment",
        name="tuple-immutable")


def test_tuple_membership_and_iteration():
    expect_output(
        "t = (1, 2, 3)\nprint(2 in t)\nprint(9 not in t)\n"
        "for x in t:\n    print(x)\n",
        ["True", "True", "1", "2", "3"],
        name="tuple-membership-iteration")


def test_tuple_equality_and_truthiness():
    expect_output(
        "t = (1, 2)\nprint(t == (1, 2))\nprint(t == (2, 1))\nprint((1, 2) != (1, 3))\n"
        "if t:\n    print(\"truthy\")\n",
        ["True", "False", "True", "truthy"],
        name="tuple-equality")


def test_unpack_implicit_tuple():
    expect_output(
        "a, b = 1, 2\nprint(a)\nprint(b)\n"
        "a, b = b, a\nprint(a)\nprint(b)\n",
        ["1", "2", "2", "1"],
        name="unpack-swap")


def test_unpack_from_iterables():
    expect_output(
        'a, b = [10, 20]\nprint(a + b)\n'
        'x, y = (30, 40)\nprint(x + y)\n'
        'c, d = "ok"\nprint(c)\nprint(d)\n',
        ["30", "70", "o", "k"],
        name="unpack-iterables")


def test_unpack_count_mismatch():
    expect_error(
        "a, b = [1, 2, 3]\n",
        "too many values to unpack",
        name="unpack-too-many")
    expect_error(
        "a, b = [1]\n",
        "not enough values to unpack",
        name="unpack-not-enough")


def test_for_unpack_dict_items():
    expect_output(
        'scores = {"alice": 90, "bob": 75}\n'
        "for k, v in scores.items():\n"
        "    print(k)\n"
        "    print(v)\n",
        ["alice", "90", "bob", "75"],
        name="for-unpack-items")


def test_for_unpack_list_of_pairs():
    expect_output(
        "pts = [(1, 2), (3, 4)]\nfor x, y in pts:\n    print(x * y)\n",
        ["2", "12"],
        name="for-unpack-pairs")


def test_tuple_in_function():
    expect_output(
        "def minmax(a, b):\n"
        "    if a < b:\n"
        "        return (a, b)\n"
        "    return (b, a)\n"
        "lo, hi = minmax(9, 3)\n"
        "print(lo)\nprint(hi)\n",
        ["3", "9"],
        name="tuple-from-function")


def test_tuple_nested_and_in_list():
    expect_output(
        't = (1, ("x", 2))\nprint(t)\nprint(t[1][0])\n'
        'pairs = [(1, "a"), (2, "b")]\nprint(pairs[0][1])\n',
        ["(1, ('x', 2))", "x", "a"],
        name="tuple-nested")


def main():
    tests = [
        ("元组字面量与打印", test_tuple_literal_and_print),
        ("元组下标与 len/type", test_tuple_index_and_len),
        ("元组下标越界报错", test_tuple_index_oob),
        ("元组不可变", test_tuple_immutable),
        ("元组成员与遍历", test_tuple_membership_and_iteration),
        ("元组相等与真值", test_tuple_equality_and_truthiness),
        ("隐式元组解包与交换", test_unpack_implicit_tuple),
        ("从 list/tuple/str 解包", test_unpack_from_iterables),
        ("解包数量不匹配报错", test_unpack_count_mismatch),
        ("for 双变量解包 items", test_for_unpack_dict_items),
        ("for 解包配对列表", test_for_unpack_list_of_pairs),
        ("函数返回元组再解包", test_tuple_in_function),
        ("嵌套元组与列表中元组", test_tuple_nested_and_in_list),
    ]
    ok = run_tests(tests)
    if not ok:
        sys.exit(1)
    print("PASS test_07_tuples")


if __name__ == "__main__":
    main()
