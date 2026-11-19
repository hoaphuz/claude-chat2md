# claude-chat2md

Convert your [Claude Code](https://docs.claude.com/en/docs/claude-code) chat history into readable Markdown files — with a one-click Windows app, no install, no runtime.

> **Unofficial community tool.** Not affiliated with, endorsed by, or sponsored by Anthropic. "Claude" is a trademark of Anthropic.

📖 Hướng dẫn sử dụng tiếng Việt: [docs/huongdan-sudung.md](docs/huongdan-sudung.md)

## Why

Claude Code stores every conversation as a `.jsonl` transcript under `~/.claude/projects`. Those files are hard to read, and when you rename or move a project folder, Claude Code no longer shows its old chats. This tool turns them into clean Markdown you can read, search, archive, or share.

- **Single portable `.exe` for Windows** — native Win32 GUI, ~2.5 MB, nothing to install (no Python, Node, or .NET).
- **Read-only** — your transcripts are never modified or deleted.
- **Whole history or just a few chats** — pick the `projects` folder, a single project folder, or one or more `.jsonl` files. The folder structure is mirrored in the output.
- **Optional details** — include tool calls with their results, and Claude's thinking, each folded inside `<details>`.
- **Incremental export** — only new or changed chats are converted; unchanged ones are skipped (tracked in `.claude-chat2md-state.json` in the output folder).
- **Readable file names** — name each `.md` after the chat's title (your custom name or the one Claude Code generates) instead of the session id.
- **Clean output** — injected system tags are stripped, subagent (sidechain) chatter is dropped, and consecutive messages from the same side are grouped.
- **Pure Go, standard library only** — no third-party dependencies, easy to audit.
- **CLI for macOS/Linux** with the same converter.

The UI is currently in Vietnamese; the button layout is simple enough to use without reading it (see the steps below).

## Download

Get the latest `claude-chat2md-portable-<version>.zip` from the [Releases](../../releases) page, unzip it anywhere you can write to, and run `claude-chat2md.exe`.

The exe is not code-signed, so Windows SmartScreen may show "Windows protected your PC". Click **More info → Run anyway**. You can compare the file with the SHA-256 checksum published in the release notes, or build it yourself (see below).

Requirements: Windows 10/11, 64-bit.

## Usage (Windows GUI)

1. **Source** (top field) — defaults to `%USERPROFILE%\.claude\projects` (or `%CLAUDE_CONFIG_DIR%\projects`). Use **Thư mục…** (Folder) to choose one project folder inside it, or **Chọn tệp…** (Choose files) to pick one or more `.jsonl` files (Ctrl/Shift-click); several paths can also be typed separated by `;`. The last used source is remembered, and the drop-down lists the 10 most recent ones. Picked files are always converted, and files inside `projects` keep their project subfolder in the output.
2. **Output** — defaults to `Documents\claude-export`. The last used folder is remembered across restarts, and the drop-down lists the 10 most recent ones (stored in `settings.json` next to the exe, or in `%APPDATA%\claude-chat2md` if that folder isn't writable).
3. Options (checkboxes, in order): include tool calls and results · include thinking · name files after the chat title (on by default; invalid characters are replaced, duplicates get ` (2)`, ` (3)`…) · only convert new or changed chats (on by default) · open the output folder when done.
4. Click the **Chuyển đổi** (Convert) button. The log box shows each exported file and a summary.

Each chat becomes one `.md` file with a title, session id, project path, start time, and timestamped turns. Converting again overwrites existing `.md` files with the same name.

> ⚠️ Exported Markdown contains the full chat — and, with tool output enabled, file contents and command output from your machine. Review it for secrets before sharing.

## Usage (CLI, macOS/Linux)

```
claude-chat2md [-tools] [-thinking] [-title] [-update] [-version] [file.jsonl | dir] [out-dir]
claude-chat2md [flags] -o out-dir file.jsonl|dir ...
```

- `-tools` — include tool calls and results
- `-thinking` — include thinking blocks
- `-title` — name files after the chat title instead of the session id
- `-update` — only convert chats that are new or changed since the last export
- `-o out-dir` — output folder; every remaining argument is then a source, so several files can be converted at once
- No source → `~/.claude/projects`; no output dir → `./claude-export`

Example: `claude-chat2md -tools ~/.claude/projects ~/Documents/claude-export`

## Build from source

Requires Go 1.21+.

```powershell
# Windows: vet + test + GUI build, copy the exe into -OutDir (overwritten in place,
# existing logs kept) + versioned portable zip in dist\
.\build.ps1
.\build.ps1 -OutDir D:\Apps\claude-chat2md   # pick another output folder
.\build.ps1 -SkipBuild -NoZip                # recopy only, no zip

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

Issues and pull requests are welcome. A short developer guide (in Vietnamese) is in [docs-dev/](docs-dev/README.md). Please keep the project dependency-free (Go standard library only) and run `go vet ./...` and `go test ./...` before submitting. When reporting a conversion bug, attach the `debug-<date>.log` file — it contains no chat content.

## License

[MIT](LICENSE)
