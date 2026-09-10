# -*- coding: utf-8 -*-
"""gopy 测试公共工具。

把内嵌的源码写入临时 .py 文件,调用 gopy 解释器执行,
对 stdout / 退出码做断言。所有测试文件共用这里的能力。
"""
import os
import subprocess
import tempfile

# 仓库根目录(gopy/)
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# 解释器可执行文件:优先 GOPY_BIN 环境变量,其次根目录的 gopy.exe / gopy
GOPY = os.environ.get("GOPY_BIN")
if not GOPY:
    for candidate in ("gopy.exe", "gopy"):
        path = os.path.join(ROOT, candidate)
        if os.path.exists(path):
            GOPY = path
            break
if not GOPY:
    raise RuntimeError("gopy 可执行文件不存在,请先在根目录执行: go build -o gopy.exe .")


def run_source(src):
    """执行一段源码,返回 (returncode, stdout, stderr)。

    固定用 LF 换行写入,便于单独测试解释器的 CRLF 兼容逻辑。
    """
    fd, path = tempfile.mkstemp(suffix=".py", prefix="gopy_test_")
    with os.fdopen(fd, "w", encoding="utf-8", newline="") as f:
        f.write(src)
    try:
        proc = subprocess.run(
            [GOPY, path],
            capture_output=True,
            text=True,
            encoding="utf-8",
            timeout=30,
        )
        return proc.returncode, proc.stdout, proc.stderr
    finally:
        os.unlink(path)


def _format_diff(name, src, expected, got):
    lines = ["[%s] 输出不匹配" % name]
    lines.append("--- 源码 ---")
    lines.extend("  " + l for l in src.strip("\n").splitlines())
    lines.append("--- 期望 ---")
    lines.extend("  " + repr(l) for l in expected.splitlines())
    lines.append("--- 实际 ---")
    lines.extend("  " + repr(l) for l in got.splitlines())
    return "\n".join(lines)


def expect_output(src, expected, name="case"):
    """断言 src 执行成功且 stdout 与 expected(字符串或行列表)一致。"""
    rc, out, err = run_source(src)
    exp = expected if isinstance(expected, str) else "\n".join(expected)
    got = out.strip("\n")
    exp = exp.strip("\n")
    if rc != 0:
        raise AssertionError(
            "[%s] 退出码=%d, stderr=%s\n%s" % (name, rc, err.strip(), got))
    if got != exp:
        raise AssertionError(_format_diff(name, src, exp, got))
    return out


def expect_error(src, fragment, name="case"):
    """断言 src 执行失败(退出码非 0)且错误信息包含 fragment。"""
    rc, out, err = run_source(src)
    if rc == 0:
        raise AssertionError(
            "[%s] 预期报错但执行成功, stdout=%s" % (name, out.strip()))
    text = (err + out).strip()
    if fragment not in text:
        raise AssertionError(
            "[%s] 错误信息不包含 %r, 实际=%s" % (name, fragment, text))
    return text


def run_tests(tests):
    """依次执行 (名称, 闭包) 列表,全部通过返回 True,否则打印失败详情。"""
    failed = 0
    for name, fn in tests:
        try:
            fn()
            print("  PASS %s" % name)
        except AssertionError as e:
            failed += 1
            print("  FAIL %s\n%s" % (name, e))
        except Exception as e:  # 环境类错误也要暴露
            failed += 1
            print("  FAIL %s (unexpected %r)" % (name, e))
    return failed == 0
