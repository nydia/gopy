# -*- coding: utf-8 -*-
"""R6: 数值语义修正 + 复合赋值 + and/or 短路。

- `/` 真除法:int/int → float(CPython 语义)
- `//` 向下取整除(负数向 -inf 取整)
- `%` 负数取模:结果符号跟随除数
- 复合赋值 += -= *= /= %= //=,支持普通变量与 arr[i] / d[k] 目标
- 列表拼接 [1]+[2] 与重复 [1]*2
- and/or 短路求值并返回操作数(不再要求 bool)
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from harness import expect_output, expect_error, run_tests


def test_true_division():
    expect_output(
        "print(7 / 2)\nprint(6 / 2)\nprint(1 / 4)\n",
        ["3.5", "3.0", "0.25"],
        name="true-division")


def test_floor_division():
    expect_output(
        "print(7 // 2)\nprint(-7 // 2)\nprint(7.5 // 2)\nprint(-7.5 // 2)\n",
        ["3", "-4", "3.0", "-4.0"],
        name="floor-division")


def test_modulo_negative():
    expect_output(
        "print(-7 % 3)\nprint(7 % -3)\nprint(7 % 3)\nprint(-7.5 % 2)\n",
        ["2", "-2", "1", "0.5"],
        name="modulo-negative")


def test_division_by_zero_errors():
    expect_error("print(1 / 0)\n", "division by zero", name="div-zero")
    expect_error("print(7 // 0)\n", "integer division or modulo by zero", name="floordiv-zero")
    expect_error("print(7 % 0)\n", "modulo by zero", name="mod-zero")


def test_float_string_format():
    # 整值 float 显示为 x.0(对齐 CPython repr)
    expect_output(
        "print(2.0)\nprint(10 / 5)\nprint(0.1 + 0.2)\n",
        ["2.0", "2.0", "0.30000000000000004"],
        name="float-format")


def test_aug_assign_basic():
    expect_output(
        "x = 5\nx += 3\nprint(x)\nx -= 1\nprint(x)\nx *= 4\nprint(x)\n"
        "x /= 2\nprint(x)\nx //= 3\nprint(x)\nx %= 3\nprint(x)\n",
        ["8", "7", "28", "14.0", "4.0", "1.0"],
        name="aug-assign-basic")


def test_aug_assign_string():
    expect_output(
        's = "a"\ns += "b"\ns *= 2\nprint(s)\n',
        "abab",
        name="aug-assign-string")


def test_aug_assign_list():
    expect_output(
        "nums = [1]\nnums += [2, 3]\nprint(nums)\nnums *= 2\nprint(nums)\n",
        ["[1, 2, 3]", "[1, 2, 3, 1, 2, 3]"],
        name="aug-assign-list")


def test_aug_assign_index_target():
    expect_output(
        "arr = [10, 20]\narr[1] += 5\nprint(arr)\n"
        'd = {"n": 1}\nd["n"] *= 6\nprint(d["n"])\n',
        ["[10, 25]", "6"],
        name="aug-assign-index-target")


def test_aug_assign_undefined():
    expect_error(
        "x += 1\n",
        "identifier not found",
        name="aug-assign-undefined")


def test_list_concat_repeat_operators():
    expect_output(
        "print([1] + [2, 3])\nprint([0] * 3)\nprint(2 * [\"a\"])\nprint([1, 2] * 0)\n",
        ["[1, 2, 3]", "[0, 0, 0]", "['a', 'a']", "[]"],
        name="list-concat-repeat")


def test_and_or_short_circuit():
    # and/or 返回操作数本身,且短路(右侧不求值)
    expect_output(
        'print(0 or "default")\nprint("x" and 42)\nprint(None and "never")\n'
        'print(3 and 0 or "fallback")\n',
        ["default", "42", "None", "fallback"],
        name="and-or-values")


def test_short_circuit_avoids_error():
    # 短路避免除零与未定义变量
    expect_output(
        "x = 0\nprint(x != 0 and 10 / x)\nprint(1 or undefined_name)\n",
        ["False", "1"],
        name="short-circuit-guard")


def test_and_or_precedence():
    # and 优先级高于 or;not 高于 and
    expect_output(
        'print(True or False and False)\nprint(not False and True)\n',
        ["True", "True"],
        name="and-or-precedence")


def main():
    tests = [
        ("真除法 /", test_true_division),
        ("整除 //", test_floor_division),
        ("负数取模", test_modulo_negative),
        ("除零报错", test_division_by_zero_errors),
        ("float 显示格式", test_float_string_format),
        ("复合赋值基础", test_aug_assign_basic),
        ("复合赋值字符串", test_aug_assign_string),
        ("复合赋值列表", test_aug_assign_list),
        ("复合赋值下标目标", test_aug_assign_index_target),
        ("复合赋值未定义变量", test_aug_assign_undefined),
        ("列表拼接与重复运算", test_list_concat_repeat_operators),
        ("and/or 返回操作数", test_and_or_short_circuit),
        ("短路避免求值错误", test_short_circuit_avoids_error),
        ("and/or 优先级", test_and_or_precedence),
    ]
    ok = run_tests(tests)
    if not ok:
        sys.exit(1)
    print("PASS test_06_numbers")


if __name__ == "__main__":
    main()
