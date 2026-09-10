# -*- coding: utf-8 -*-
"""运行 test/ 下所有 test_*.py 测试文件,输出汇总。

用法: 在仓库任意位置执行  python test/run_all.py
"""
import os
import subprocess
import sys

TEST_DIR = os.path.dirname(os.path.abspath(__file__))


def main():
    files = sorted(
        f for f in os.listdir(TEST_DIR)
        if f.startswith("test_") and f.endswith(".py")
    )
    if not files:
        print("没有找到测试文件")
        return 1

    print("gopy 测试套件: %d 个文件\n" % len(files))
    failed = []
    for f in files:
        print("== %s" % f)
        proc = subprocess.run(
            [sys.executable, os.path.join(TEST_DIR, f)],
            capture_output=True, text=True, encoding="utf-8",
        )
        sys.stdout.write(proc.stdout)
        if proc.stderr:
            sys.stdout.write(proc.stderr)
        if proc.returncode != 0:
            failed.append(f)
        print("")

    total = len(files)
    ok = total - len(failed)
    print("=" * 40)
    print("结果: %d/%d 通过" % (ok, total))
    if failed:
        print("失败: %s" % ", ".join(failed))
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
