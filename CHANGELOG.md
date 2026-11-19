# Changelog

## 0.10.0 – 2026-11-19
- Nút **?** ở góc trên bên phải (hoặc phím F1, hoặc mục "Giới thiệu và hướng dẫn…" trong menu biểu tượng góc trái thanh tiêu đề) mở hộp giới thiệu chương trình.
- Hộp giới thiệu có link mở tài liệu hướng dẫn sử dụng và trang dự án trên GitHub.

## 0.9.0 – 2026-11-19
- Không còn ghi đè file `.md` có sẵn trùng tên mà không phải bản xuất của chat đó (ghi chú tự tạo, file của công cụ khác…): chat được đặt tên mới có `(2)`, `(3)`…
- Lỡ xóa file trạng thái `.claude-chat2md-state.json` thì các bản xuất cũ vẫn được nhận ra (qua dòng Session) và ghi đè đúng chỗ, không sinh bản trùng.
- Nút mới **Thư mục VS Code…** để chỉ định bằng tay nơi đọc nhóm chat (hữu ích với VS Code portable đang tắt); báo ngay số nhóm đọc được.

## 0.8.0 – 2026-11-19
- Hỗ trợ nhóm chat tạo trong VS Code (khung Session Manager → New group). Tùy chọn mới "Ghi tên nhóm chat vào đầu tệp" (mặc định bật) thêm dòng **Nhóm** vào đầu file.
- Tùy chọn mới "Xếp tệp vào thư mục con theo nhóm chat" (mặc định tắt): mỗi nhóm một thư mục con trong thư mục dự án.
- Đổi nhóm của chat thì lần chuyển sau file được dời sang đúng chỗ, file cũ (chưa sửa) và thư mục nhóm trống được dọn.
- Tự tìm dữ liệu VS Code, kể cả bản portable đang mở. Bản dòng lệnh dùng cờ `-group`, `-group-dirs`, `-vscode`.

## 0.7.0 – 2026-11-19
- Ô dữ liệu chat (nguồn) và ô thư mục lưu tệp Markdown đều nhớ giá trị dùng lần cuối, mở lại phần mềm không còn về mặc định.
- Nút sổ xuống ở hai ô này hiện 10 mục dùng gần đây để chọn lại nhanh.
- Cài đặt lưu ở file `settings.json` ngay cạnh file exe (cùng chỗ với thư mục `logs`), mang cả thư mục đi máy khác vẫn giữ nguyên.
- Đổi chữ trên giao diện: nút **Chọn tệp…**, "Thư mục lưu tệp Markdown", "Đặt tên tệp theo tên phiên chat".

## 0.6.0 – 2026-11-19
- Nút mới **Chọn file…**: chọn một hoặc nhiều file chat (giữ Ctrl/Shift) để chuyển riêng, không cần chuyển cả thư mục. Ô nguồn cũng nhận nhiều đường dẫn cách nhau bằng `;`.
- File chọn riêng luôn được chuyển, và được đặt vào thư mục con theo dự án giống khi chuyển cả thư mục `projects`, nên cập nhật đúng file đã xuất trước đó.
- Bản dòng lệnh: cờ `-o <thư-mục-đầu-ra>` để chuyển nhiều file cùng lúc.

## 0.5.0 – 2026-11-19
- Tùy chọn mới "Chỉ chuyển chat mới hoặc đã thay đổi" (mặc định bật): chat đã xuất và không đổi được bỏ qua, chuyển lại nhanh hơn nhiều. Bản dòng lệnh dùng cờ `-update`.
- Khi tên phiên đổi, file Markdown tên cũ được tự xóa (trừ khi bạn đã sửa file đó).
- Dòng tổng kết hiện thêm số chat không đổi.

## 0.4.0 – 2026-11-19
- Tùy chọn mới "Đặt tên file theo tên phiên chat" (mặc định bật): file `.md` mang tên phiên thay vì mã phiên ngẫu nhiên. Bản dòng lệnh dùng cờ `-title`.
- Tiêu đề file Markdown nay lấy cả tên Claude Code tự đặt cho phiên (khi bạn chưa tự đặt tên).
- Hai phiên trùng tên trong cùng thư mục được thêm `(2)`, `(3)`… để không ghi đè lẫn nhau.

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
