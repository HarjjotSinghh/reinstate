# conptydriver

The Windows twin of `scripts/testing/vendor-tty-driver.py`: runs a command
under a real Windows pseudo console (ConPTY), drives it with a small step
script, and renders what actually appeared through a real VT screen model —
not a regex strip of the raw bytes.

Built from `golang.org/x/sys/windows` (`CreatePseudoConsole`,
`ResizePseudoConsole`, `ClosePseudoConsole`, the `ProcThreadAttributeList`
API). No other new module dependency. Windows only: on any other `GOOS` the
package still builds (so `go build ./...` cross-compiles cleanly for darwin
and linux, per the release gates), but every run returns
`"ConPTY is only available on Windows"`.

## Usage

```powershell
# from the repository root
.\scripts\testing\conptydriver\conptydriver.ps1 -cols 80 -rows 24 -script steps.txt -- .\bin\rein.exe
```

```bash
# Git Bash, from the repository root
./scripts/testing/conptydriver/conptydriver.sh -cols 80 -rows 24 -script steps.txt -- ./bin/rein.exe
```

Or directly:

```
conptydriver [-cols 80] [-rows 25] [-dir WORKDIR] [-script steps.txt] [-raw raw.log] [-exit-timeout 15s] [-bg 'rgb:0000/0000/0000'] -- <command> [args...]
```

| Flag | Meaning |
| ---- | ------- |
| `-cols`, `-rows` | pseudo console size |
| `-dir` | child working directory (default: this process's) |
| `-script` | a step-script file (grammar below); omitted runs an interactive, line-buffered passthrough instead |
| `-raw` | also write every raw byte the child produced (VT sequences and all) to this file |
| `-exit-timeout` | how long to wait for the child to exit after the script finishes before killing it |
| `-bg` | the `rgb:RRRR/GGGG/BBBB` string this console answers an OSC 11 (background colour) query with |

Everything after `--` is the command and its arguments; `-cols`/`-rows`/etc.
must come before `--`.

## Step-script grammar

One verb per line; blank lines and lines starting with `#` are ignored.

```
wait /regex/ 10s        block until the rendered frame matches regex, or
                        fail (naming the source line and the last frame) if
                        the timeout passes first
send "text"             write text to the child's input, literally; Go
                        string-literal escapes apply (\n, \", \\, ...)
key enter|esc|tab|ctrl+k|up|down|left|right|space|backspace|<char>
                        write one named keystroke; ctrl+X (X a-z) sends the
                        control byte for letter X; a single character not
                        otherwise named is sent literally, so `key f` and
                        `send "f"` are the same thing
snapshot path           render the current frame and write it to path
                        (parent directories created as needed)
sleep 500ms              an unconditional pause, for a step with no signal
                        worth waiting on
kill                     terminate the child immediately
```

`wait` matches against the **rendered** frame (after cursor moves, erases,
and SGR are applied), not the raw byte stream — see the first trap below.

Example, driving `scripts/tuisandbox`'s bare `rein` to its first frame:

```
wait /ctrl\+k commands/ 15s
snapshot switcher-frame.txt
kill
```

## The VT screen model

`vtscreen.go` is a small, purpose-built terminal model: a grid of cells and
a cursor, updated by CUP/CUU/CUD/CUF/CUB (`H`/`f`, `A`, `B`, `C`, `D`), CHA/VPA
(`G`, `d`), CR, LF, backspace, EL (`K`), ED (`J`); SGR (`m`) is parsed and
discarded (colour plays no part in a snapshot); OSC/DCS/SOS/PM/APC strings
are consumed to their terminator and otherwise ignored, except OSC 11
(background-colour query) and CSI 6n (cursor-position report), which are
answered — see the second trap below. It has its own unit tests
(`vtscreen_test.go`) independent of any real pseudo console.

### Trap 1 — conhost rewrites spaces as cursor-forward moves

conhost's own repaint strategy frequently redraws a run of *unchanged*
cells as a cursor-forward move (`ESC [ n C`) instead of literal space
bytes. A regex strip of "ANSI-looking" runs cannot tell that apart from
nothing having happened, and either drops those columns or double-counts
them depending on which way it guesses. A real cursor model does not need
to guess: `CUF` moves the cursor without touching cell content, and every
cell starts (and is cleared to) a space, so the columns a `CUF` skips over
already read as blank — which is what they are. `TestScreenCursorForwardLeavesBlanks`
is this trap as a test.

### Trap 2 — Bubble Tea's startup queries

A Bubble Tea program (Reinstate's interactive CLI included) sends two
device queries before its first real frame and blocks on their replies:
`ESC ] 11 ; ? BEL` (what is the terminal's background colour — light or
dark theme?) and `ESC [ 6 n` (where is the cursor?). A driver that does not
answer these leaves the program hanging forever, waiting for a reply no
real terminal here will ever send. `vtscreen.go` answers both; `-bg` picks
what the background-colour answer says.

## What this proves, and what it does not

`CreatePseudoConsole` → `ProcThreadAttributeList` (`PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE`)
→ `CreateProcess` with `EXTENDED_STARTUPINFO_PRESENT` is the sequence
Microsoft's own ConPTY sample uses; `conpty_windows.go` follows it exactly,
including calling `UpdateProcThreadAttribute` through a small raw-syscall
wrapper (`updateProcThreadAttributePseudoConsole`) rather than through
`ProcThreadAttributeListContainer.Update`, because that method needs the
attribute value as `unsafe.Pointer` and this attribute's value is the
`HPCON` handle itself — a plain integer with no Go pointer behind it, so
there is no sound way to route it through `unsafe.Pointer` that `go vet`'s
`unsafeptr` check will not (correctly, for every case except this
documented Win32 contract) flag as a possible misuse.

On the acceptance host this was built and verified on, that sequence
allocates a pseudo console, runs a child inside it, and the child's raw
output stream (real VT sequences: console-init, OSC title changes,
cursor-visibility toggles) comes back correctly — see
`docs/testing/windows-acceptance-host.md`'s "ConPTY driver" and "Results"
sections for the exact evidence, including a full run driving
`scripts/tuisandbox`'s bare `rein` to its switcher and a real Claude Code
`--resume` to a completed response.

### One trap: `conptydriver.exe` needs a real console of its own

`CreatePseudoConsole`'s "attach the child to a new console" step inherits
whatever console `conptydriver.exe` itself has. Launch `conptydriver.exe`
from something that redirects **its** stdout/stderr to a pipe — many
automation tools do this by default to capture output — and the pseudo
console it allocates for its own child is real (openable and correctly
VT-moded through `CONOUT$`/`CONIN$` from inside the child) but
`GetStdHandle` in the grandchild does not resolve to it, so any
`golang.org/x/term`-style `IsTerminal` check (which `rein`'s bare switcher
and most Bubble Tea programs gate their first frame on) reports false and
the child refuses to run interactively.

Run `conptydriver.exe` from a real interactive PowerShell/Windows Terminal
session (the ordinary case) and this never comes up. Driving it from an
automated context that itself captures output: use PowerShell's
`Start-Process` **without** `-RedirectStandardOutput`/
`-RedirectStandardError` (`-WindowStyle Hidden` is fine — it only skips
showing the window, it does not redirect stdio) so `conptydriver.exe` gets
a genuine console, and have `conptydriver.exe`'s own `-script`/`-raw`/
`-snapshot` flags write results to files instead of relying on its stdout
being captured. If a script's `wait` step never matches and the child
immediately says it needs a terminal, this is the first thing to check.
