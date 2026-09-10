# -*- coding: utf-8 -*-
"""R4: 方法调用语法(obj.method(...))与列表/字典方法。

- 新增 `.` 属性访问 + BoundMethod(接收者自动作为第一个参数)
- list: append pop insert extend clear copy index count reverse sort
- dict: keys values items get clear
- 方法可作为值传递(f = nums.append)
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from harness import expect_output, expect_error, run_tests


def test_list_append_pop():
    expect_output(
        "nums = [1, 2]\nnums.append(3)\nprint(nums)\n"
        "x = nums.pop()\nprint(x)\nprint(nums)\n"
        "y = nums.pop(0)\nprint(y)\nprint(nums)\n",
        ["[1, 2, 3]", "3", "[1, 2]", "1", "[2]"],
        name="list-append-pop")


def test_list_insert_extend_clear():
    expect_output(
        "a = [1, 3]\na.insert(1, 2)\nprint(a)\n"
        "a.insert(-10, 0)\nprint(a)\n"
        "b = a.copy()\nb.extend([7, 8])\nprint(b)\n"
        "b.clear()\nprint(b)\nprint(len(b))\n",
        ["[1, 2, 3]", "[0, 1, 2, 3]", "[0, 1, 2, 3, 7, 8]", "[]", "0"],
        name="list-insert-extend-clear")


def test_list_index_count():
    expect_output(
        "a = [5, 3, 5, 1]\nprint(a.index(5))\nprint(a.count(5))\nprint(a.count(9))\n",
        ["0", "2", "0"],
        name="list-index-count")


def test_list_index_not_found():
    expect_error(
        "a = [1, 2]\nprint(a.index(9))\n",
        "not in list",
        name="list-index-missing")


def test_list_reverse_sort():
    expect_output(
        "a = [3, 1, 2]\na.reverse()\nprint(a)\n"
        "a.sort()\nprint(a)\n"
        's = ["banana", "apple", "cherry"]\ns.sort()\nprint(s)\n',
        ["[2, 1, 3]", "[1, 2, 3]", "['apple', 'banana', 'cherry']"],
        name="list-reverse-sort")


def test_list_sort_mixed_types_error():
    expect_error(
        'a = [1, "two"]\na.sort()\n',
        "not supported",
        name="list-sort-mixed")


def test_dict_keys_values_items():
    expect_output(
        'd = {"b": 2, "a": 1}\nprint(d.keys())\nprint(d.values())\nprint(d.items())\n',
        ["['b', 'a']", "[2, 1]", "[('b', 2), ('a', 1)]"],
        name="dict-keys-values-items")


def test_dict_get():
    expect_output(
        'd = {"a": 1}\nprint(d.get("a"))\nprint(d.get("zz"))\nprint(d.get("zz", 99))\n',
        ["1", "None", "99"],
        name="dict-get")


def test_dict_items_iteration():
    expect_output(
        'd = {"x": 10, "y": 20}\n'
        "for pair in d.items():\n"
        "    print(pair[0])\n"
        "    print(pair[1])\n",
        ["x", "10", "y", "20"],
        name="dict-items-iteration")


def test_dict_clear():
    expect_output(
        'd = {"a": 1}\nd.clear()\nprint(d)\nprint(len(d))\n',
        ["{}", "0"],
        name="dict-clear")


def test_method_as_value():
    # 方法是一等公民:可赋值后调用,接收者已绑定
    expect_output(
        "nums = [1]\nf = nums.append\nf(2)\nf(3)\nprint(nums)\n",
        "[1, 2, 3]",
        name="method-as-value")


def test_unknown_method_errors():
    expect_error(
        "nums = [1]\nnums.nothing()\n",
        "has no attribute",
        name="list-unknown-method")
    expect_error(
        's = "abc"\ns.append(1)\n',
        "has no attribute",
        name="str-no-method-yet")
    expect_error(
        "x = 3\nx.foo()\n",
        "has no attribute",
        name="int-no-attr")


def test_method_on_expression_receiver():
    # 接收者可以是任意表达式,方法返回值可以继续链式调用
    expect_output(
        "print([3, 1, 2].copy().reverse())\n"
        "nums = [1, 2]\nnums.append(3)\nprint(nums)\n",
        ["None", "[1, 2, 3]"],
        name="method-expression-receiver")


def main():
    tests = [
        ("list append/pop", test_list_append_pop),
        ("list insert/extend/clear/copy", test_list_insert_extend_clear),
        ("list index/count", test_list_index_count),
        ("list index 未找到报错", test_list_index_not_found),
        ("list reverse/sort", test_list_reverse_sort),
        ("list sort 混合类型报错", test_list_sort_mixed_types_error),
        ("dict keys/values/items", test_dict_keys_values_items),
        ("dict get 默认值", test_dict_get),
        ("dict items 遍历", test_dict_items_iteration),
        ("dict clear", test_dict_clear),
        ("方法作为值传递", test_method_as_value),
        ("未知方法/对象报错", test_unknown_method_errors),
        ("接收者可以是表达式", test_method_on_expression_receiver),
    ]
    ok = run_tests(tests)
    if not ok:
        sys.exit(1)
    print("PASS test_04_methods")


if __name__ == "__main__":
    main()
