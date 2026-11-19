# Hướng dẫn cho người phát triển

Tài liệu này dành cho người phát triển. Hướng dẫn cho người dùng cuối nằm ở [../docs/](../docs/huongdan-sudung.md).
Chi tiết kiến trúc và quy ước kỹ thuật (chủ yếu để Claude đọc) nằm trong [../CLAUDE.md](../CLAUDE.md).

## Thao tác thường dùng

```powershell
go run .                 # chạy thử, log vào dev\logs
go vet ./...             # kiểm tra tĩnh
go test ./...            # chạy test
.\build.ps1              # vet + test + build + chép exe vào OutDir + zip vào dist\
.\build.ps1 -SkipBuild -NoZip    # chỉ chép lại exe đã build
go run ./tools/mkrsrc    # sinh lại icon exe, sau khi sửa assets\icons\png\app-icon.ico
```

Cần Go ≥ 1.21 và PowerShell 5.1. Không cần cài thêm gì khác.

## Làm việc với Claude

- Cứ mô tả việc cần làm; Claude tự đọc `CLAUDE.md` để biết quy tắc của dự án.
- Claude sẽ chạy `go vet` và `go test` trước khi báo xong.
- Đổi hành vi hoặc giao diện thì Claude tăng `VERSION`, thêm mục vào `CHANGELOG.md` và sửa `docs/huongdan-sudung.md` (kể cả số phiên bản trong đó) và `README.md` nếu cần. Nếu thấy thiếu, nhắc Claude.

## Lưu ý quan trọng

- **Chỉ dùng thư viện chuẩn của Go**, không cgo, không thêm thư viện bên thứ ba.
- **Không thêm cờ `-w`** vào lệnh build GUI: phần mềm diệt virus trên máy sẽ chặn file linker (`Access is denied`). Dùng đúng `-trimpath -ldflags "-H=windowsgui -s"`.
- File `.ps1` phải lưu **UTF-8 có BOM**, nếu không tiếng Việt bị lỗi.
- **Giao diện Win32 chỉ được gọi từ luồng chính.** Goroutine nền gửi kết quả về giao diện qua `postEvent`, không gọi trực tiếp.
- **Không ghi nội dung chat vào log**, chỉ ghi đường dẫn, số lượng, thời gian, lỗi.
- **Không sửa hay xóa transcript nguồn.** Công cụ chỉ đọc.
- Dự án công khai (MIT): không đưa đường dẫn hoặc tên cá nhân, nơi làm việc vào file được git theo dõi; dùng ví dụ chung như `D:\Projects\my-app`. Ghi chú riêng để trong `tasks/` (đã gitignore).
- Test của lõi nằm trong `convert_test.go`; khi sửa lõi nên thêm test dùng `tempDir(t)` (thay cho `t.TempDir()`, chịu được antivirus giữ file khi dọn; giả lập thư mục projects bằng `t.Setenv("CLAUDE_CONFIG_DIR", …)`), không đưa transcript thật vào repo.
- **Dòng Session ở đầu file `.md` là "chữ ký" của bản xuất:** khi trùng tên mà file không có trong file trạng thái, ứng dụng đọc dòng này để biết có được ghi đè không. Đổi định dạng dòng này thì phải sửa `reSessionLine` trong `convert.go`.
- Hộp thoại chọn thư mục/file không chạy được trong test tự động; sau khi sửa phần giao diện, chạy `go run .` và bấm thử. Màn hình phóng to (DPI > 96) nên kiểm tra kỹ chữ có bị che không: tọa độ trong code tính ở 96 DPI rồi nhân tỉ lệ.

## Nhóm chat của VS Code

Tóm tắt những điều đã tìm hiểu khi làm tính năng nhóm (bản 0.8.0):

- **Nhóm không nằm trong file `.jsonl`.** Extension Claude Code lưu nhóm trong globalState của VS Code, tức file SQLite `User\globalStorage\state.vscdb` (mỗi profile có thêm một file ở `User\profiles\<id>\globalStorage\state.vscdb`). Trong bảng `ItemTable`, một dòng có giá trị JSON chứa các khóa `sessionGroups:<thư mục dự án>` = `[{id, name, collapsed, sessionIds}]`. `sessionIds` chính là tên file `.jsonl`.
- **Thư mục dữ liệu VS Code:** bản cài thường là `%APPDATA%\Code` (macOS `~/Library/Application Support/Code`, Linux `~/.config/Code`). Bản portable là `<thư mục cài>\data\user-data`, chỉ dò được khi VS Code đang chạy, nên GUI lưu lại vào `settings.json` (`vscodeDirs`). Người dùng cũng có thể thêm bằng tay qua nút "Thư mục VS Code…".
- **VS Code ghi globalState xuống đĩa trễ** (khoảng vài chục giây đến khi đóng). Đổi nhóm xong mà đọc ngay có thể còn dữ liệu cũ.
- **Vì sao tự viết bộ đọc SQLite:** `mattn/go-sqlite3` cần cgo; `modernc.org/sqlite` thì Go thuần nhưng kéo theo nhiều gói và làm exe to thêm nhiều MB. Ta chỉ cần đọc một bảng, nên `sqlite.go` (vài trăm dòng) là đủ.
- **Không tìm chữ thẳng trong file `.vscdb`:** các trang trống vẫn còn bản cũ (từng thấy cả tên nhóm "New group" trước khi đổi tên). Phải duyệt đúng cây B-tree như `sqlite.go` đang làm.
- Hiện VS Code dùng chế độ journal thường (byte 18–19 của header = 1). `sqlite.go` vẫn đọc được chế độ WAL (file `-wal`, chỉ lấy khung đã commit) phòng khi VS Code đổi.
- **Khi điều tra file thật:** chép `state.vscdb` ra chỗ khác trước khi đọc, không mở file đang dùng. Thử bộ đọc với bản chép: `$env:CHAT2MD_VSCDB="<bản chép>"; go test -run TestReadRealStateDB -v .`
- **Test:** máy không có Python/sqlite3 để tạo file mẫu, nên `vscode_test.go` có bộ ghi SQLite tối giản (nhiều trang lá, trang trong, trang tràn, WAL). Test gọi `isolateVSCode` để không đọc dữ liệu VS Code thật trên máy.
- **Dọn file cũ khi đổi nhóm:** file trạng thái lưu thêm `groupDir`, để phân biệt thư mục nhóm với thư mục dự án. Nếu không, xuất riêng một dự án vào thư mục đã có bản xuất cả `projects` sẽ xóa nhầm file (có test `TestGroupsKeepOtherBaseExport`).
- Nếu extension đổi cách lưu nhóm (đổi tên khóa, cấu trúc JSON), sửa `parseGroupsValue` trong `vscode.go`. Chương trình chỉ ghi cảnh báo "Không tìm thấy nhóm chat nào" và vẫn xuất bình thường.

- Hộp **Giới thiệu** (nút ?, F1) dẫn tới tài liệu hướng dẫn trên GitHub (`guideURL` trong `about_windows.go`). Đổi tên repo, nhánh chính hoặc đường dẫn `docs/huongdan-sudung.md` thì sửa hằng số này.

## Khi cần gỡ lỗi

- Xem `dev\logs\debug-<ngày>.log` khi `go run .`; với bản đã build, bấm nút "Mở thư mục log".
- Người dùng báo lỗi chuyển đổi: nhờ họ gửi file `debug-<ngày>.log` (không chứa nội dung chat).
- Mất icon ứng dụng trong Explorer: chạy `go run ./tools/mkrsrc` rồi build lại.
- Không ra nhóm chat: trong `debug-<ngày>.log` có dòng `Read N session groups from <file>` cho từng `state.vscdb` đã đọc, và `Running portable VS Code user-data dirs` cho bản portable dò được.

## Phát hành

1. Tăng `VERSION`, cập nhật `CHANGELOG.md` và các tài liệu liên quan.
2. Chạy `.\build.ps1`, mở thử exe vừa build.
3. Commit, tạo Release trên GitHub, đính kèm file zip trong `dist\` và ghi SHA-256 của exe vào ghi chú phát hành.
