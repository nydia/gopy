# -*- coding: utf-8 -*-
"""R10: try/except/raise 异常机制。

- try/except 捕获运行时错误;裸 except 捕获一切
- 类型化 except:ZeroDivisionError / KeyError / IndexError / NameError / ValueError
- raise ValueError("msg") 抛出;as e 绑定错误消息
- 未捕获异常冒泡到顶层退出;循环/函数跨层传播
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from harness import expect_output, expect_error, run_tests


def test_try_bare_except():
    expect_output(
        "try:\n    x = 1 / 0\nexcept:\n    print(\"caught\")\nprint(\"after\")\n",
        ["caught", "after"],
        name="try-bare-except")


def test_typed_zero_division():
    expect_output(
        "try:\n    x = 1 / 0\nexcept ZeroDivisionError as e:\n    print(e)\n",
        ["division by zero"],
        name="typed-zero-division")


def test_typed_key_error():
    expect_output(
        'd = {"a": 1}\ntry:\n    print(d["missing"])\nexcept KeyError as e:\n    print(e)\nprint("ok")\n',
        ["missing", "ok"],
        name="typed-key-error")


def test_typed_index_error():
    expect_output(
        "nums = [1, 2]\ntry:\n    t = nums[5]\nexcept IndexError:\n    print(\"oob\")\n",
        ["oob"],
        name="typed-index-error")


def test_typed_name_error():
    expect_output(
        "try:\n    y = undefined_var + 1\nexcept NameError:\n    print(\"no name\")\n",
        ["no name"],
        name="typed-name-error")


def test_raise_and_catch():
    expect_output(
        'def check(n):\n    if n < 0:\n        raise ValueError("negative: " + str(n))\n    return n\n'
        'try:\n    check(-5)\nexcept ValueError as e:\n    print(e)\n',
        ["negative: -5"],
        name="raise-and-catch")


def test_wrong_handler_propagates():
    expect_error(
        'try:\n    raise ValueError("boom")\nexcept KeyError:\n    print("never")\n',
        "boom",
        name="wrong-handler-propagates")


def test_uncaught_exits():
    expect_error(
        'raise RuntimeError("fatal")\n',
        "fatal",
        name="uncaught-exits")


def test_try_in_loop():
    expect_output(
        "nums = [1, 0, 2]\nfor n in nums:\n    try:\n        print(10 / n)\n    except ZeroDivisionError:\n        print(\"skip zero\")\n",
        ["10.0", "skip zero", "5.0"],
        name="try-in-loop")


def test_exception_across_calls():
    expect_output(
        "def inner():\n    return 1 / 0\n"
        "def outer():\n    return inner()\n"
        "try:\n    outer()\nexcept ZeroDivisionError:\n    print(\"crossed frames\")\n",
        ["crossed frames"],
        name="exception-across-calls")


def test_no_exception_skips_handler():
    expect_output(
        "try:\n    x = 2 + 3\nexcept:\n    print(\"never\")\nprint(x)\n",
        ["5"],
        name="no-exception")


def test_multiple_except_clauses():
    expect_output(
        'd = {"k": 1}\ntry:\n    print(d["zz"])\nexcept IndexError:\n    print("idx")\nexcept KeyError as e:\n    print(e)\n',
        ["zz"],
        name="multiple-except")


def test_raise_string():
    expect_output(
        'try:\n    raise "plain string error"\nexcept:\n    print("caught string")\n',
        ["caught string"],
        name="raise-string")


def main():
    tests = [
        ("裸 except 捕获一切", test_try_bare_except),
        ("ZeroDivisionError", test_typed_zero_division),
        ("KeyError", test_typed_key_error),
        ("IndexError", test_typed_index_error),
        ("NameError", test_typed_name_error),
        ("raise 与捕获", test_raise_and_catch),
        ("不匹配处理器继续冒泡", test_wrong_handler_propagates),
        ("未捕获异常退出", test_uncaught_exits),
        ("循环内 try/except", test_try_in_loop),
        ("异常跨函数帧传播", test_exception_across_calls),
        ("无异常跳过处理器", test_no_exception_skips_handler),
        ("多个 except 子句", test_multiple_except_clauses),
        ("raise 字符串", test_raise_string),
    ]
    ok = run_tests(tests)
    if not ok:
        sys.exit(1)
    print("PASS test_10_exceptions")


if __name__ == "__main__":
    main()
