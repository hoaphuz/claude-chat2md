# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`claude-chat2md` converts Claude Code chat transcripts (`.jsonl` files under `~/.claude/projects`, or `$CLAUDE_CONFIG_DIR/projects`) into Markdown. Pure Go, standard library only, no cgo, no third-party dependencies — keep it that way.

## Rules

- Shell is PowerShell 5.1 on Windows. `.ps1` files must be UTF-8 with BOM.
- GUI build flags are `-trimpath -ldflags "-H=windowsgui -s"`. Do not add `-w`: on this machine that exact combo makes the antivirus block the linker output ("a.out.exe: Access is denied"); `-s` alone gives the same size.
- Before reporting a task as done, `go vet ./...` and `go test ./...` must pass.
- When a change affects commands, architecture, conventions or these rules, update CLAUDE.md in the same task. Keep it concise.
- The user guide is `docs/huongdan-sudung.md` (Vietnamese, for end users). When UI or behavior it describes changes, update it in the same task, including the version it states.
- The developer guide is the single short file `docs-dev/README.md` (Vietnamese, for humans: common commands, working with Claude, key cautions, release). Keep it separate from `docs/`, which is end-user only, and keep it short: architecture and detailed conventions belong here in CLAUDE.md, not there. Update it when commands or the key rules change.
- The project is public (MIT, `LICENSE`). `README.md` is the English GitHub page; keep its features, options and CLI usage in sync with the app. Never put personal or workplace paths/names in tracked files; use generic examples like `D:\Projects\my-app`. `tasks/` holds private work notes and is gitignored.

## Versioning & changelog

- Single source of version: the `VERSION` file (semver), embedded by `version.go` (`appVersion`, shown in the window title and by CLI `-version`) and read by `build.ps1`.
- After each task that changes app behavior or UI, bump the version (fix → patch, feature/UI → minor) and prepend a Vietnamese entry to `CHANGELOG.md` (`## x.y.z – YYYY-MM-DD`, 1–5 user-facing bullets). Read only its first ~20 lines. Do this last, after vet/test pass.

## Commands

```powershell
go run .                           # dev: opens the GUI (plus a console); logs go to the project's dev\logs
go run ./tools/mkrsrc              # regenerate rsrc_windows_amd64.syso (app icon) from assets\icons\png\app-icon.ico
.\build.ps1                        # mkrsrc + vet + test + GUI build (build\claude-chat2md.exe), copy exe over -OutDir (default D:\Program\Portable\claude-chat2md\build, no per-version dirs, keeps logs) + versioned zip in dist\
.\build.ps1 -SkipBuild -NoZip      # repackage only, no zip
go build -ldflags "-H=windowsgui -s" -o claude-chat2md.exe   # quick Windows GUI build
$env:GOOS="linux"; go build -o claude-chat2md; Remove-Item Env:GOOS   # macOS/Linux CLI cross-build
go vet ./...
go test ./...
go test -run TestParseVersion .   # single test
```

CLI usage (non-Windows): `claude-chat2md [-tools] [-thinking] [-title] [-update] [-version] [file.jsonl | dir] [out-dir]`, or `claude-chat2md [flags] -o out-dir file.jsonl|dir ...` for several sources.

## Architecture

Single `package main`, split by build tag:

- `convert.go` — OS-independent core. `ConvertAll` takes a list of inputs (files and/or directories; the GUI source field holds them `;`-separated, see `SplitInputs`/`JoinInputs`), walks directories (skipping `subagents`, `memory`, `tool-results`), dedupes, calls `ConvertFile` per `.jsonl`, and mirrors the relative path into the output dir as `.md`. A directory's files are relative to that directory; an explicitly listed file is relative to `DefaultSourceDir()` if inside it (keeps the project subfolder), else placed flat (`fileBase`), and is always converted even when unchanged. Each file goes through `safeConvert`, so a panic fails only that file. The core never touches UI; it reports only via the logger.
- `state.go` — `.claude-chat2md-state.json` in the output dir: per source (abs path key) its size+mtime, `formatKey` (version + content/naming options) and the `.md` written (with its size+mtime). `ConvertAll` reserves each source's previous `.md` name, skips unchanged sources when `Options.Incremental` is set (`Result.Unchanged`), always rewrites the state, and `removeStale` deletes a source's old `.md` in the same dir after a rename only if the user hasn't edited it.
- `settings.go` — GUI settings remembered across runs in `settings.json`, located like the logs (`settingsPath`, chosen once per run): project `dev\` under `go run` → next to the exe (portable) → `%APPDATA%\claude-chat2md` if the exe dir isn't writable. Holds the last source (`LastSrc`, may be a `;`-joined file list) and output dir (`LastOut`), each with up to 10 recent values (`RecentSrc`/`RecentOut`, `addRecent` dedupes case-insensitively). Both GUI fields are editable `COMBOBOX`es (`CBS_DROPDOWN`, built by `historyCombo`) listing them; `remember` saves on pick, on convert, and on close.
- `logger.go` — two tiers. `logInfo/logWarn/logError`: short Vietnamese lines sent to the UI sink (`setLogSink`) and to `logs\<date>.log`. `logDebug`: English details for debugging, only in `logs\debug-<date>.log` (which also gets every UI line). Log dir (`findLogDir`): project `dev\logs` when the exe sits in a `go-build` dir (`go run`) → `logs` next to the exe → `%LOCALAPPDATA%\claude-chat2md\logs`. Before `initLogger` (e.g. in tests) nothing is written to disk. `cleanOldLogs` deletes only files named like the app's own logs, older than 7 days. Never log chat content; only paths, sizes, counts (`convStats`), timings, errors.
- `main_windows.go` (`//go:build windows`) — native Win32 GUI written directly against `user32`/`gdi32`/`shell32`/`comctl32`/`comdlg32` via `syscall.LazyDLL`. Folder picker: `SHBrowseForFolderW`; file picker: `GetOpenFileNameW` with multi-select (`pickFiles`, result parsed by `parseMultiSelect`). Key points:
  - `runtime.LockOSThread()` in `init` — all Win32 calls must stay on the main thread.
  - Conversion runs in a goroutine. Log lines and the final `done` event go through `postEvent` (buffered channel + `PostMessage(wmAppEvent)`), and `drainEvents` applies them on the UI thread. Never call UI functions from the worker goroutine.
  - Visual styles are enabled by writing a Common Controls v6 manifest to the temp dir and activating it at runtime (no embedded resource/manifest).
  - Layout uses fixed 96-DPI coordinates scaled through `scale()`.
  - Icons: design set lives in `assets/icons/` (svg + png, light/dark). The app icon is embedded as a PE resource via `rsrc_windows_amd64.syso` (RT_GROUP_ICON id 1), generated from `assets/icons/png/app-icon.ico` by `go run ./tools/mkrsrc` (stdlib-only COFF writer; `build.ps1` reruns it). Regenerate the `.syso` whenever the `.ico` changes. Button icons are the `light` PNGs, `go:embed`ded and turned into HICONs with `CreateIconFromResourceEx` (`@2x` at ≥144 DPI).
- `main_other.go` (`//go:build !windows`) — minimal `flag`-based CLI calling the same `ConvertAll`.

### Transcript parsing rules (in `ConvertFile`)

- Each line is an `entry`; malformed lines are skipped (counted in the debug log).
- Title priority: `custom-title` entry > last `ai-title` entry (`aiTitle`, Claude Code's auto name) > first `summary` entry > first line of the first user message (80 runes) > session filename. `ConvertFile` returns it; with `Options.TitleName`, `ConvertAll` names the file `safeFileName(title)` (falls back to the session id) and `uniqueName` adds ` (2)`… on clashes within a run.
- Sidechain (subagent), `isMeta`, and non-`user`/`assistant` entries are dropped.
- `message.content` may be a string or an array of blocks (`parseContent`). A `user` message consisting only of `tool_result` blocks is merged into Claude's turn rather than starting a user turn.
- User text is stripped of injected tags (`<system-reminder>`, `<command-*>`, `<local-command-stdout>`) by `cleanUserText`.
- Consecutive messages from the same role share one `### role` header. Tool calls/results and thinking are emitted only when the matching `Options` flag is set, wrapped in `<details>`; tool output is truncated to `maxToolChars` runes, and `fence` picks a backtick fence longer than any run inside the content.
- Source transcripts are read-only; the tool never modifies or deletes them.

## Conventions

- Code comments, UI strings, UI-tier log messages (`logInfo/Warn/Error`), errors returned to the UI, and generated Markdown labels are in Vietnamese. `logDebug` messages and identifiers are in English.
