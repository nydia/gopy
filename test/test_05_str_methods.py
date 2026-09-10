# -*- coding: utf-8 -*-
"""R5: 字符串方法与字符串下标。

- upper lower capitalize strip/lstrip/rstrip replace split join
  startswith endswith find count isdigit
- 字符串下标 s[i](含负数,越界报 IndexError)
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from harness import expect_output, expect_error, run_tests


def test_case_methods():
    expect_output(
        'print("hello".upper())\nprint("WORLD".lower())\nprint("hello world".capitalize())\n',
        ["HELLO", "world", "Hello world"],
        name="str-case")


def test_strip_family():
    expect_output(
        'print("  hi  ".strip())\n'
        'print("xxhixx".strip("x"))\n'
        'print("  hi  ".lstrip())\n'
        'print("  hi  ".rstrip())\n',
        ["hi", "hi", "hi  ", "  hi"],
        name="str-strip-family")


def test_replace():
    expect_output(
        'print("aXbXc".replace("X", "-"))\nprint("aaa".replace("a", "bb"))\nprint("abc".replace("z", "-"))\n',
        ["a-b-c", "bbbbbb", "abc"],
        name="str-replace")


def test_split():
    expect_output(
        'print("a,b,c".split(","))\n'
        'print("a b  c".split())\n'
        'print("a,b,,c".split(","))\n'
        'print("abc".split())\n',
        ["['a', 'b', 'c']", "['a', 'b', 'c']", "['a', 'b', '', 'c']", "['abc']"],
        name="str-split")


def test_split_empty_separator_error():
    expect_error(
        'print("abc".split(""))\n',
        "empty separator",
        name="str-split-empty-sep")


def test_join():
    expect_output(
        'print("-".join(["a", "b", "c"]))\nprint("".join(["x", "y"]))\n',
        ["a-b-c", "xy"],
        name="str-join")


def test_join_non_string_error():
    expect_error(
        'print("-".join(["a", 1]))\n',
        "expected str instance",
        name="str-join-non-str")


def test_startswith_endswith():
    expect_output(
        'print("hello.py".startswith("hello"))\nprint("hello.py".startswith("py"))\n'
        'print("hello.py".endswith(".py"))\nprint("hello.py".endswith(".js"))\n',
        ["True", "False", "True", "False"],
        name="str-starts-ends")


def test_find_count():
    expect_output(
        'print("hello".find("ll"))\nprint("hello".find("z"))\n'
        'print("ababab".count("ab"))\nprint("aaa".count("aa"))\n',
        ["2", "-1", "3", "1"],
        name="str-find-count")


def test_isdigit():
    expect_output(
        'print("123".isdigit())\nprint("12a".isdigit())\nprint("".isdigit())\n',
        ["True", "False", "False"],
        name="str-isdigit")


def test_string_indexing():
    expect_output(
        'print("abc"[0])\nprint("abc"[-1])\nprint("abc"[1])\n',
        ["a", "c", "b"],
        name="str-index")


def test_string_index_oob():
    expect_error(
        'print("abc"[5])\n',
        "index out of range",
        name="str-index-oob")


def test_chained_methods():
    expect_output(
        'print("  Hello World  ".strip().upper())\nprint("a,b".split(",").count("a"))\n',
        ["HELLO WORLD", "1"],
        name="str-chained")


def test_str_in_dict_value():
    expect_output(
        'd = {"name": " gopy "}\nprint(d["name"].strip())\n',
        "gopy",
        name="str-method-on-dict-value")


def main():
    tests = [
        ("大小写方法", test_case_methods),
        ("strip 家族", test_strip_family),
        ("replace", test_replace),
        ("split", test_split),
        ("split 空分隔符报错", test_split_empty_separator_error),
        ("join", test_join),
        ("join 非字符串报错", test_join_non_string_error),
        ("startswith/endswith", test_startswith_endswith),
        ("find/count", test_find_count),
        ("isdigit", test_isdigit),
        ("字符串下标", test_string_indexing),
        ("字符串下标越界报错", test_string_index_oob),
        ("链式方法调用", test_chained_methods),
        ("对 dict 值调用方法", test_str_in_dict_value),
    ]
    ok = run_tests(tests)
    if not ok:
        sys.exit(1)
    print("PASS test_05_str_methods")


if __name__ == "__main__":
    main()
