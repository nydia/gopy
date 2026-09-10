# -*- coding: utf-8 -*-
"""R1: 词法器对 CRLF(Windows 换行)的兼容。

Windows 上 checkout 出来的源码通常是 \r\n 换行,解释器必须能正确处理:
- 语句结尾的 \r\n 当作一个换行
- 行首缩进后跟 \r\n 的空行要跳过
- 注释行的 \r\n 要跳过
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from harness import expect_output, expect_error, run_tests


def test_crlf_basic():
    # 语句用 CRLF 结尾
    expect_output(
        "x = 1\r\ny = 2\r\nprint(x + y)\r\n",
        "3",
        name="crlf-basic")


def test_crlf_indent_blocks():
    # 缩进块 + CRLF
    expect_output(
        "if True:\r\n    print(\"in block\")\r\n    print(\"still in\")\r\nprint(\"after\")\r\n",
        ["in block", "still in", "after"],
        name="crlf-indent-block")


def test_crlf_blank_and_comment_lines():
    # 空行、纯空白行、注释行都带 \r\n
    src = (
        "for i in range(3):\r\n"
        "\r\n"
        "    # 注释行\r\n"
        "    print(i)\r\n"
        "\r\n"
        "print(\"done\")\r\n"
    )
    expect_output(src, ["0", "1", "2", "done"], name="crlf-blank-comment")


def test_crlf_function_def():
    # def + return + CRLF
    src = (
        "def add(a, b):\r\n"
        "    return a + b\r\n"
        "\r\n"
        "print(add(2, 3))\r\n"
        "print(add(10, -1))\r\n"
    )
    expect_output(src, ["5", "9"], name="crlf-def")


def test_crlf_nested_dedent():
    # 多级 DEDENT 出现在 CRLF 文件中
    src = (
        "if True:\r\n"
        "    if False:\r\n"
        "        print(\"no\")\r\n"
        "    print(\"inner\")\r\n"
        "print(\"outer\")\r\n"
    )
    expect_output(src, ["inner", "outer"], name="crlf-nested-dedent")


def test_lf_still_works():
    # 修复不能破坏原有 LF 行为
    expect_output(
        "x = [1, 2, 3]\nprint(len(x))\nprint(x[-1])\n",
        ["3", "3"],
        name="lf-regression")


def main():
    tests = [
        ("CRLF 基本语句", test_crlf_basic),
        ("CRLF 缩进块", test_crlf_indent_blocks),
        ("CRLF 空行/注释行", test_crlf_blank_and_comment_lines),
        ("CRLF 函数定义", test_crlf_function_def),
        ("CRLF 多级 DEDENT", test_crlf_nested_dedent),
        ("LF 回归", test_lf_still_works),
    ]
    ok = run_tests(tests)
    if not ok:
        sys.exit(1)
    print("PASS test_01_crlf")


if __name__ == "__main__":
    main()
