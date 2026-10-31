# claude-chat2md

Convert your [Claude Code](https://docs.claude.com/en/docs/claude-code) chat history into readable Markdown files — with a one-click Windows app, no install, no runtime.

> **Unofficial community tool.** Not affiliated with, endorsed by, or sponsored by Anthropic. "Claude" is a trademark of Anthropic.

📖 Hướng dẫn sử dụng tiếng Việt: [docs/huongdan-sudung.md](docs/huongdan-sudung.md)

## Why

Claude Code stores every conversation as a `.jsonl` transcript under `~/.claude/projects`. Those files are hard to read, and when you rename or move a project folder, Claude Code no longer shows its old chats. This tool turns them into clean Markdown you can read, search, archive, or share.

- **Single portable `.exe` for Windows** — native Win32 GUI, ~2.5 MB, nothing to install (no Python, Node, or .NET).
- **Read-only** — your transcripts are never modified or deleted.
- **Whole history or one chat** — pick the `projects` folder, a single project folder, or one `.jsonl` file. The folder structure is mirrored in the output.
- **Optional details** — include tool calls with their results, and Claude's thinking, each folded inside `<details>`.
- **Clean output** — injected system tags are stripped, subagent (sidechain) chatter is dropped, and consecutive messages from the same side are grouped.
- **Pure Go, standard library only** — no third-party dependencies, easy to audit.
- **CLI for macOS/Linux** with the same converter.

The UI is currently in Vietnamese; the button layout is simple enough to use without reading it (see the steps below).

## Download

Get the latest `claude-chat2md-portable-<version>.zip` from the [Releases](../../releases) page, unzip it anywhere you can write to, and run `claude-chat2md.exe`.

The exe is not code-signed, so Windows SmartScreen may show "Windows protected your PC". Click **More info → Run anyway**. You can compare the file with the SHA-256 checksum published in the release notes, or build it yourself (see below).

Requirements: Windows 10/11, 64-bit.

## Usage (Windows GUI)

1. **Source** (top field) — defaults to `%USERPROFILE%\.claude\projects` (or `%CLAUDE_CONFIG_DIR%\projects`). You can also choose one project folder inside it, or type the path of a single `.jsonl` file.
2. **Output** — defaults to `Documents\claude-export`.
3. Options (checkboxes, in order): include tool calls and results · include thinking · open the output folder when done.
4. Click the **Chuyển đổi** (Convert) button. The log box shows each exported file and a summary.

Each chat becomes one `.md` file with a title, session id, project path, start time, and timestamped turns. Converting again overwrites existing `.md` files with the same name.

> ⚠️ Exported Markdown contains the full chat — and, with tool output enabled, file contents and command output from your machine. Review it for secrets before sharing.

## Usage (CLI, macOS/Linux)

```
claude-chat2md [-tools] [-thinking] [-version] [file.jsonl | dir] [out-dir]
```

- `-tools` — include tool calls and results
- `-thinking` — include thinking blocks
- No source → `~/.claude/projects`; no output dir → `./claude-export`

Example: `claude-chat2md -tools ~/.claude/projects ~/Documents/claude-export`

## Build from source

Requires Go 1.21+.

```powershell
# Windows: vet + test + GUI build + portable zip in dist\
.\build.ps1

# Quick Windows GUI build
go build -trimpath -ldflags "-H=windowsgui -s" -o claude-chat2md.exe

# macOS / Linux CLI
go build -o claude-chat2md
```

## Logs & privacy

The app writes logs to a `logs` folder next to the exe (fallback: `%LOCALAPPDATA%\claude-chat2md\logs`), deleted automatically after 7 days. Logs contain only paths, counts, timings, and errors — **never chat content**. The tool makes no network requests.

## Similar projects

If you prefer a CLI, HTML output, or a web viewer, check out [claude-code-log](https://github.com/daaain/claude-code-log), [cc2md](https://github.com/magarcia/cc2md), [claude-code-chat-browser](https://github.com/CppDigest/claude-code-chat-browser), or [claude-JSONL-browser](https://github.com/withLinda/claude-JSONL-browser). This project focuses on a zero-install, double-click experience for Windows users.

## Contributing

Issues and pull requests are welcome. Please keep the project dependency-free (Go standard library only) and run `go vet ./...` and `go test ./...` before submitting. When reporting a conversion bug, attach the `debug-<date>.log` file — it contains no chat content.

## License

[MIT](LICENSE)
