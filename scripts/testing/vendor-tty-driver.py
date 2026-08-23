#!/usr/bin/env python3
"""Run a vendor CLI on a pseudo-terminal that answers its startup queries.

Acceptance rows that launch a vendor TUI need a real terminal on both stdin and
stdout. A passive recorder is not enough: a Bubble Tea style TUI asks the
terminal for its device attributes, cursor position and colours before it draws
anything, and blocks until each is answered. Under script(1) the process stays
alive with zero bytes captured, which is indistinguishable from a hung vendor
and has cost more than one wasted acceptance run.

This driver replies to those queries, records everything the vendor wrote, and
can inject input when a pattern appears -- which is how an approval prompt gets
answered and evidenced in the same transcript.

    vendor-tty-driver.py --log run.log --timeout 900 -- grok --session-id UUID "prompt"
    vendor-tty-driver.py --log run.log --expect '\\(y/n\\)' --send 'y\\n' -- grok ...

Exit status is the vendor's own, or 124 if the timeout was reached.
"""

from __future__ import annotations

import argparse
import fcntl
import os
import pty
import re
import select
import signal
import struct
import sys
import termios
import time

# Queries a TUI issues at startup, and the answer a real terminal would give.
# Without these the vendor never proceeds to its first frame.
_XTVERSION = (re.compile(rb"\x1b\[>0?q"), b"\x1bP>|vendor-tty-driver(1)\x1b\\")
_SECONDARY_DA = (re.compile(rb"\x1b\[>c|\x1b\[>0c"), b"\x1b[>0;95;0c")
_STATUS = (re.compile(rb"\x1b\[5n"), b"\x1b[0n")
_DEVICE_ATTRS = (re.compile(rb"\x1b\[c|\x1b\[0c"), b"\x1b[?1;2c")
_CURSOR_POS = (re.compile(rb"\x1b\[6n"), b"\x1b[24;80R")
_COLOURS = [
    (re.compile(rb"\x1b\]10;\?(?:\x07|\x1b\\)"), b"\x1b]10;rgb:ffff/ffff/ffff\x07"),
    (re.compile(rb"\x1b\]11;\?(?:\x07|\x1b\\)"), b"\x1b]11;rgb:0000/0000/0000\x07"),
]


def main() -> int:
    ap = argparse.ArgumentParser(add_help=True)
    ap.add_argument("--log", required=True, help="write the raw vendor transcript here")
    ap.add_argument("--timeout", type=float, default=600.0, help="seconds before giving up")
    ap.add_argument("--expect", action="append", default=[], metavar="REGEX",
                    help="pattern to wait for; pair with --send (repeatable, matched in order)")
    ap.add_argument("--send", action="append", default=[], metavar="TEXT",
                    help=r"input to write when the matching --expect fires; \n and \t are decoded")
    ap.add_argument("--stop-on", metavar="REGEX", help="terminate once this appears")
    ap.add_argument("--rows", type=int, default=40, help="terminal height reported to the vendor")
    ap.add_argument("--cols", type=int, default=120, help="terminal width reported to the vendor")
    ap.add_argument("cmd", nargs=argparse.REMAINDER, help="-- then the vendor command")
    args = ap.parse_args()

    cmd = args.cmd[1:] if args.cmd and args.cmd[0] == "--" else args.cmd
    if not cmd:
        ap.error("no command given; put it after --")
    if len(args.expect) != len(args.send):
        ap.error("each --expect needs exactly one --send")

    pending = [
        (re.compile(e.encode(), re.I), s.encode().decode("unicode_escape").encode())
        for e, s in zip(args.expect, args.send)
    ]
    stop_on = re.compile(args.stop_on.encode(), re.I) if args.stop_on else None

    pid, fd = pty.fork()
    if pid == 0:
        os.environ.setdefault("TERM", "xterm-256color")
        os.execvp(cmd[0], cmd)
        os._exit(127)

    # A pty with no window size is 0x0, and a TUI given 0x0 draws nothing but
    # its title -- which reads as a hang for the same reason an unanswered
    # query does. Set a real size before the vendor's first frame.
    try:
        fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", args.rows, args.cols, 0, 0))
    except OSError:
        pass

    os.set_blocking(fd, False)
    deadline = time.time() + args.timeout
    status, window = None, b""

    def answer(data: bytes) -> None:
        try:
            os.write(fd, data)
        except OSError:
            pass

    with open(args.log, "wb") as log:
        while time.time() < deadline:
            try:
                readable, _, _ = select.select([fd], [], [], 0.5)
            except OSError:
                break
            if fd in readable:
                try:
                    chunk = os.read(fd, 65536)
                except OSError:
                    break
                if not chunk:
                    break
                log.write(chunk)
                log.flush()
                window += chunk

                for pattern, reply in [_DEVICE_ATTRS, _SECONDARY_DA, _XTVERSION,
                                       _CURSOR_POS, _STATUS, *_COLOURS]:
                    if pattern.search(window):
                        answer(reply)
                        window = pattern.sub(b"", window)

                # Injections fire in order, once each, so a prompt that repeats
                # does not consume an answer meant for a later one.
                if pending and pending[0][0].search(window):
                    _, text = pending.pop(0)
                    answer(text)

                if stop_on and stop_on.search(window):
                    break
                # Bounded, but long enough that a query split across reads is
                # still matched whole.
                window = window[-8192:]

            done, raw = os.waitpid(pid, os.WNOHANG)
            if done == pid:
                status = raw
                break

    if status is None:
        for sig in (signal.SIGTERM, signal.SIGKILL):
            try:
                os.kill(pid, sig)
                _, status = os.waitpid(pid, 0)
                break
            except (ProcessLookupError, ChildProcessError):
                status = 0
                break
        else:
            status = 0
        if time.time() >= deadline:
            return 124

    return os.waitstatus_to_exitcode(status) if status else 0


if __name__ == "__main__":
    sys.exit(main())
