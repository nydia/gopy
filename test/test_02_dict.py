# -*- coding: utf-8 -*-
"""R2: 字典 dict。

覆盖:字面量(含空 dict)、下标读、下标写(新增/覆盖)、len、
for-in 遍历 keys(插入序)、缺失键报错、真值判断、type()、嵌套结构、
以及 for-in 对字符串的逐字符遍历。
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from harness import expect_output, expect_error, run_tests


def test_dict_literal_and_print():
    expect_output(
        'd = {"name": "Alice", "age": 30}\nprint(d)\nprint(len(d))\n',
        ["{'name': 'Alice', 'age': 30}", "2"],
        name="dict-literal")


def test_dict_empty():
    expect_output(
        "d = {}\nprint(d)\nprint(len(d))\nif not d:\n    print(\"empty is falsy\")\n",
        ["{}", "0", "empty is falsy"],
        name="dict-empty")


def test_dict_read():
    expect_output(
        'd = {"a": 1, 2: "two"}\nprint(d["a"])\nprint(d[2])\n',
        ["1", "two"],
        name="dict-read")


def test_dict_read_missing_key():
    expect_error(
        'd = {"a": 1}\nprint(d["missing"])\n',
        "KeyError",
        name="dict-missing-key")


def test_dict_write_insert_and_update():
    expect_output(
        'd = {"a": 1}\nd["b"] = 2\nd["a"] = 100\nprint(d)\nprint(len(d))\n',
        ["{'a': 100, 'b': 2}", "2"],
        name="dict-write")


def test_dict_for_in_keys_insertion_order():
    expect_output(
        'd = {"x": 1, "y": 2, "z": 3}\nfor k in d:\n    print(k)\n',
        ["x", "y", "z"],
        name="dict-for-in")


def test_dict_for_in_sum_values():
    expect_output(
        'd = {"a": 1, "b": 2, "c": 3}\ntotal = 0\nfor k in d:\n    total = total + d[k]\nprint(total)\n',
        "6",
        name="dict-sum-values")


def test_dict_truthiness_nonempty():
    expect_output(
        'd = {"a": 1}\nif d:\n    print("truthy")\n',
        "truthy",
        name="dict-truthy")


def test_dict_nested():
    expect_output(
        'users = {"alice": {"score": 90}, "bob": {"score": 75}}\n'
        'print(users["alice"]["score"])\n'
        'users["bob"]["score"] = 80\n'
        'print(users["bob"]["score"])\n',
        ["90", "80"],
        name="dict-nested")


def test_dict_type_and_int_keys_distinct():
    expect_output(
        'd = {1: "int-one", "1": "str-one"}\nprint(d[1])\nprint(d["1"])\nprint(type(d))\n',
        ["int-one", "str-one", "dict"],
        name="dict-int-vs-str-keys")


def test_dict_in_function():
    expect_output(
        'def counter_init(n):\n'
        '    return {"count": n}\n'
        'c = counter_init(5)\n'
        'c["count"] = c["count"] + 1\n'
        'print(c["count"])\n',
        "6",
        name="dict-in-function")


def test_string_iteration_chars():
    # for-in 顺带支持字符串逐字符遍历
    expect_output(
        'for c in "abc":\n    print(c)\n',
        ["a", "b", "c"],
        name="str-iteration")


def main():
    tests = [
        ("字典字面量与打印", test_dict_literal_and_print),
        ("空字典与真值", test_dict_empty),
        ("下标读取", test_dict_read),
        ("缺失键报 KeyError", test_dict_read_missing_key),
        ("下标写入新增/覆盖", test_dict_write_insert_and_update),
        ("for-in 按插入序遍历键", test_dict_for_in_keys_insertion_order),
        ("for-in 累加字典值", test_dict_for_in_sum_values),
        ("非空字典为真", test_dict_truthiness_nonempty),
        ("嵌套字典读写", test_dict_nested),
        ("int 键与 str 键不混淆 + type", test_dict_type_and_int_keys_distinct),
        ("字典在函数/闭包中使用", test_dict_in_function),
        ("字符串逐字符 for-in", test_string_iteration_chars),
    ]
    ok = run_tests(tests)
    if not ok:
        sys.exit(1)
    print("PASS test_02_dict")


if __name__ == "__main__":
    main()
