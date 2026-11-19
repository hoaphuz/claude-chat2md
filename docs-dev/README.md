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
- Test của lõi nằm trong `convert_test.go`; khi sửa lõi nên thêm test dùng `t.TempDir()` (giả lập thư mục projects bằng `t.Setenv("CLAUDE_CONFIG_DIR", …)`), không đưa transcript thật vào repo.
- Hộp thoại chọn thư mục/file không chạy được trong test tự động; sau khi sửa phần giao diện, chạy `go run .` và bấm thử.

## Khi cần gỡ lỗi

- Xem `dev\logs\debug-<ngày>.log` khi `go run .`; với bản đã build, bấm nút "Mở thư mục log".
- Người dùng báo lỗi chuyển đổi: nhờ họ gửi file `debug-<ngày>.log` (không chứa nội dung chat).
- Mất icon ứng dụng trong Explorer: chạy `go run ./tools/mkrsrc` rồi build lại.

## Phát hành

1. Tăng `VERSION`, cập nhật `CHANGELOG.md` và các tài liệu liên quan.
2. Chạy `.\build.ps1`, mở thử exe vừa build.
3. Commit, tạo Release trên GitHub, đính kèm file zip trong `dist\` và ghi SHA-256 của exe vào ghi chú phát hành.
