# Changelog

## 0.3.1 – 2026-10-31
- Đổi tên ứng dụng thành `claude-chat2md` (file exe, gói portable, bản dòng lệnh). Thư mục log dự phòng chuyển sang `%LOCALAPPDATA%\claude-chat2md\logs`.

## 0.3.0 – 2026-10-25
- Bộ biểu tượng mới: icon ứng dụng hiện trên file exe, thanh tiêu đề và thanh tác vụ.
- Các nút (Chọn…, Chuyển đổi, Mở thư mục đầu ra, Mở thư mục log) có icon minh họa.

## 0.2.0 – 2026-10-20
- Ghi nhật ký ra file trong thư mục `logs`: file `<ngày>.log` giống ô Nhật ký trên giao diện, và `debug-<ngày>.log` chi tiết để sửa lỗi (chỉ ghi đường dẫn, số liệu, thời gian; không chép nội dung chat).
- Nút mới "Mở thư mục log"; file log tự xóa sau 7 ngày.
- Một file chat bị lỗi bất thường không còn làm dừng cả lượt chuyển đổi.

## 0.1.1 – 2026-10-20
- Đổi tên ứng dụng thành `claude-vscode-chat-to-md` (file exe, gói portable, bản dòng lệnh).

## 0.1.0 – 2026-10-20
- Chuyển transcript chat Claude Code (`.jsonl` trong `.claude\projects`) sang Markdown, giữ nguyên cấu trúc thư mục; dữ liệu gốc chỉ được đọc.
- Tùy chọn kèm các lệnh Claude đã chạy cùng kết quả, và phần suy luận (thinking), đều thu gọn trong `<details>`.
- Giao diện Windows: chọn thư mục nguồn/đầu ra, xem nhật ký, tự mở thư mục đầu ra khi xong.
- Bản dòng lệnh cho macOS/Linux (`-tools`, `-thinking`, `-version`).
- Số phiên bản hiện trên tiêu đề cửa sổ.
