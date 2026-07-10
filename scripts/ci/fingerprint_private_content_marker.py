#!/usr/bin/env python3
"""Create a non-plaintext marker fingerprint for the boundary policy."""

from __future__ import annotations

import getpass
import hashlib


def main() -> int:
    marker = getpass.getpass("Private/business marker (input hidden): ").strip().lower()
    if not marker or any(ord(character) > 127 for character in marker):
        raise SystemExit("marker must be non-empty ASCII")
    data = marker.encode("ascii")
    rolling = 0
    for byte in data:
        rolling = ((rolling * 257) + byte) & ((1 << 64) - 1)
    print(f"{len(data)} {rolling:016x} {hashlib.sha256(data).hexdigest()}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
