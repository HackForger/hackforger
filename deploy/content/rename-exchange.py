#!/usr/bin/env python3
"""Fail-closed Linux renameat2(RENAME_EXCHANGE) helper.

Production activation uses this helper so the fixed live directory is never
absent. There is deliberately no non-atomic fallback here.
"""

from __future__ import annotations

import ctypes
import errno
import os
import shutil
import sys
import tempfile
import stat


AT_FDCWD = -100
RENAME_EXCHANGE = 2


def fail(message: str, code: int = 1) -> "None":
    print(f"FATAL: {message}", file=sys.stderr)
    raise SystemExit(code)


def rename_exchange(left: str, right: str) -> None:
    if sys.platform != "linux":
        fail("rename exchange is supported only on Linux")
    if not os.path.isdir(left) or os.path.islink(left):
        fail(f"exchange operand is not a real directory: {left}")
    if not os.path.isdir(right) or os.path.islink(right):
        fail(f"exchange operand is not a real directory: {right}")
    if os.stat(left).st_dev != os.stat(right).st_dev:
        fail("exchange operands are not on the same filesystem")

    libc = ctypes.CDLL(None, use_errno=True)
    renameat2 = getattr(libc, "renameat2", None)
    if renameat2 is None:
        fail("libc does not expose renameat2; refusing a non-atomic fallback")
    renameat2.argtypes = [
        ctypes.c_int,
        ctypes.c_char_p,
        ctypes.c_int,
        ctypes.c_char_p,
        ctypes.c_uint,
    ]
    renameat2.restype = ctypes.c_int
    result = renameat2(
        AT_FDCWD,
        os.fsencode(left),
        AT_FDCWD,
        os.fsencode(right),
        RENAME_EXCHANGE,
    )
    if result != 0:
        error = ctypes.get_errno()
        detail = os.strerror(error) if error else "unknown error"
        fail(f"renameat2(RENAME_EXCHANGE) failed: {detail} (errno={error})")

    if os.environ.get("CONTENT_TEST_RENAME_FAIL_AFTER_SYSCALL") == "1":
        fail("injected failure after renameat2 syscall", 70)

    parents = {os.path.dirname(os.path.abspath(left)), os.path.dirname(os.path.abspath(right))}
    for parent in parents:
        try:
            descriptor = os.open(parent, os.O_RDONLY | getattr(os, "O_DIRECTORY", 0))
            try:
                os.fsync(descriptor)
            finally:
                os.close(descriptor)
        except OSError as exc:
            if exc.errno not in (errno.EINVAL, errno.ENOTSUP, errno.EROFS):
                raise


def fsync_path(path: str) -> None:
    metadata = os.lstat(path)
    if stat.S_ISLNK(metadata.st_mode):
        fail(f"refusing to fsync a symbolic link: {path}")
    if not (stat.S_ISREG(metadata.st_mode) or stat.S_ISDIR(metadata.st_mode)):
        fail(f"refusing to fsync a special inode: {path}")
    flags = os.O_RDONLY
    if stat.S_ISDIR(metadata.st_mode):
        flags |= getattr(os, "O_DIRECTORY", 0)
    descriptor = os.open(path, flags)
    try:
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


def fsync_tree(root: str) -> None:
    if not os.path.isdir(root) or os.path.islink(root):
        fail(f"durable tree root is not a real directory: {root}")
    directories = []
    for current, dirnames, filenames in os.walk(root, topdown=True, followlinks=False):
        directories.append(current)
        for name in dirnames:
            candidate = os.path.join(current, name)
            if os.path.islink(candidate):
                fail(f"durable tree contains a symbolic link: {candidate}")
        for name in filenames:
            candidate = os.path.join(current, name)
            fsync_path(candidate)
    for directory in reversed(directories):
        fsync_path(directory)


def probe(parent: str) -> None:
    if not os.path.isdir(parent) or os.path.islink(parent):
        fail(f"probe parent is not a real directory: {parent}")
    probe_root = tempfile.mkdtemp(prefix=".rename-exchange-probe-", dir=parent)
    left = os.path.join(probe_root, "left")
    right = os.path.join(probe_root, "right")
    try:
        os.mkdir(left)
        os.mkdir(right)
        with open(os.path.join(left, "left-marker"), "wb") as handle:
            handle.write(b"left\n")
        with open(os.path.join(right, "right-marker"), "wb") as handle:
            handle.write(b"right\n")
        rename_exchange(left, right)
        if not os.path.isfile(os.path.join(left, "right-marker")):
            fail("rename exchange probe did not move the right directory atomically")
        if not os.path.isfile(os.path.join(right, "left-marker")):
            fail("rename exchange probe did not move the left directory atomically")
        rename_exchange(left, right)
        if not os.path.isfile(os.path.join(left, "left-marker")):
            fail("rename exchange probe could not restore the original order")
    finally:
        shutil.rmtree(probe_root, ignore_errors=True)


def main() -> None:
    if len(sys.argv) == 3 and sys.argv[1] == "probe":
        probe(sys.argv[2])
        return
    if len(sys.argv) == 4 and sys.argv[1] == "exchange":
        rename_exchange(sys.argv[2], sys.argv[3])
        return
    if len(sys.argv) >= 3 and sys.argv[1] == "fsync":
        for path in sys.argv[2:]:
            fsync_path(path)
        return
    if len(sys.argv) == 3 and sys.argv[1] == "fsync-tree":
        fsync_tree(sys.argv[2])
        return
    fail("usage: rename-exchange.py probe PARENT | exchange LEFT RIGHT | fsync PATH... | fsync-tree ROOT", 64)


if __name__ == "__main__":
    main()
