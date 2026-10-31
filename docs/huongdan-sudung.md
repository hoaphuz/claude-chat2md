# Hướng dẫn sử dụng claude-chat2md

## Giới thiệu

claude-chat2md chuyển các đoạn chat với Claude Code (trong VS Code hoặc terminal) thành file Markdown để đọc lại, lưu trữ, tìm kiếm hoặc chia sẻ. Mỗi đoạn chat thành một file `.md`, có tiêu đề, thời gian và lượt nói của bạn và Claude.

Nên dùng khi:

- Muốn lưu lại cách đã giải quyết một vấn đề cùng Claude để xem lại sau.
- Sắp đổi tên hoặc chuyển thư mục dự án và muốn giữ một bản chat dễ đọc.
- Cần gửi nội dung một đoạn chat cho người khác xem.

Công cụ **chỉ đọc** dữ liệu chat của Claude Code, không sửa hay xóa gì trong đó.

Yêu cầu máy: Windows 10 hoặc 11 bản 64-bit. Không cần cài thêm gì. Trên macOS/Linux có bản dòng lệnh (xem cuối tài liệu).

Tài liệu này viết cho phiên bản 0.3.1. Số phiên bản hiện trên tiêu đề cửa sổ.

## Cài đặt và mở công cụ

Công cụ là bản portable, không cần cài đặt.

1. Giải nén file `claude-chat2md-portable-<phiên bản>.zip` vào một thư mục bạn có quyền ghi, ví dụ `D:\Tools\claude-chat2md`.
2. Chạy `claude-chat2md.exe`.
3. Nếu Windows hiện cảnh báo "Windows protected your PC" (file exe chưa ký số), bấm **More info** → **Run anyway**.

## Chuyển chat sang Markdown

1. **Thư mục dữ liệu chat của Claude Code.** Mặc định là `%USERPROFILE%\.claude\projects`, nơi Claude Code lưu mọi đoạn chat. Thường không cần đổi. Có thể chọn:
    - Cả thư mục `projects`: chuyển toàn bộ chat của mọi dự án.
    - Một thư mục con bên trong: chỉ chuyển chat của một dự án (xem mục *Tìm chat của một dự án*).
    - Gõ thẳng đường dẫn một file `.jsonl`: chỉ chuyển đúng đoạn chat đó.
2. **Thư mục lưu file Markdown.** Mặc định là `Documents\claude-export`. Bấm **Chọn…** để đổi.
3. **Chọn tùy chọn** (xem bảng dưới).
4. **Bấm "Chuyển đổi"**. Ô Nhật ký hiện từng file đã xuất và dòng tổng kết, ví dụ `Xong: 63 file Markdown, bỏ qua 0 file trống, 0 lỗi.`
5. Nếu đã đánh dấu "Mở thư mục đầu ra khi xong", thư mục kết quả tự mở. Lúc khác có thể bấm **Mở thư mục đầu ra**.

| Tùy chọn | Tác dụng |
| --- | --- |
| Kèm các lệnh Claude đã chạy và kết quả | Thêm các lần Claude đọc file, chạy lệnh, sửa code… cùng kết quả. Mỗi mục thu gọn, bấm để mở. Kết quả quá 3000 ký tự được cắt bớt. Bỏ chọn thì file chỉ còn lời trao đổi, ngắn và dễ đọc hơn. |
| Kèm phần suy luận (thinking) của Claude | Thêm phần Claude suy nghĩ trước khi trả lời (nếu có), cũng thu gọn. |
| Mở thư mục đầu ra khi xong | Tự mở thư mục kết quả sau khi chuyển xong. Mặc định bật. |

## File Markdown nhận được

Thư mục đầu ra giữ nguyên cấu trúc của thư mục nguồn: mỗi dự án một thư mục con, mỗi đoạn chat một file đặt theo mã phiên, ví dụ:

```
claude-export\
  d--Projects-my-app\
    c258cffe-93d4-4b44-80c5-31ba05815f96.md
```

Mỗi file gồm:

- **Tiêu đề**: tên bạn đã đặt cho đoạn chat; nếu không có thì lấy tóm tắt của Claude Code, hoặc dòng đầu tiên bạn gõ.
- **Session** (mã phiên), **Thư mục dự án** và **Bắt đầu** (giờ máy bạn).
- Các lượt **👤 Bạn** và **🤖 Claude**, mỗi lượt có giờ gửi. Nhiều tin liên tiếp của cùng một bên được gộp dưới một tiêu đề.

Những gì **không** có trong file:

- Các đoạn hệ thống tự chèn vào tin nhắn (nhắc nhở, nội dung lệnh `/` bên trong). Lệnh `/` bạn gõ vẫn giữ tên lệnh, ví dụ `/init`.
- Hội thoại của các agent phụ (subagent) do Claude tự tạo.
- Hình ảnh: chỉ hiện chữ *[hình ảnh đính kèm]*.

Đoạn chat không có tin nhắn nào sẽ bị bỏ qua và được tính vào "bỏ qua … file trống".

Xem file đẹp nhất bằng VS Code (Ctrl+Shift+V để xem trước) hoặc bất kỳ trình đọc Markdown nào.

## Tìm chat của một dự án

Claude Code đặt tên thư mục chat theo đường dẫn dự án, thay các ký tự `:`, `\`, dấu cách, dấu chấm… bằng dấu `-`. Ví dụ dự án `D:\Projects\my-app` có chat nằm trong:

```
%USERPROFILE%\.claude\projects\d--Projects-my-app
```

Chọn đúng thư mục này làm thư mục nguồn để chỉ xuất chat của dự án đó.

**Khi đổi tên hoặc chuyển thư mục dự án**, Claude Code sẽ không còn hiện các đoạn chat cũ của dự án (dữ liệu vẫn còn, chỉ nằm ở thư mục mang tên cũ). Nên xuất ra Markdown trước khi đổi tên. Nếu muốn Claude Code vẫn thấy lịch sử, hãy đóng VS Code rồi đổi tên luôn thư mục chat trong `.claude\projects` cho khớp với đường dẫn mới.

## Lưu ý quan trọng

- **Chuyển lại sẽ ghi đè.** Mỗi lần bấm "Chuyển đổi", file `.md` trùng tên trong thư mục đầu ra bị ghi đè. Nếu đã tự sửa hoặc ghi chú vào file Markdown, hãy chép sang chỗ khác hoặc chọn thư mục đầu ra khác.
- **Xem lại trước khi chia sẻ.** File Markdown chứa nguyên nội dung chat; khi bật "Kèm các lệnh Claude đã chạy" còn có cả nội dung file, kết quả lệnh, đường dẫn trên máy. Kiểm tra để không lộ mật khẩu, token, dữ liệu nội bộ.
- Chat đang diễn ra vẫn chuyển được; file Markdown chứa nội dung tới thời điểm bấm "Chuyển đổi".

## Nhật ký (log)

Công cụ ghi nhật ký vào thư mục `logs` cạnh file exe (nếu không ghi được ở đó thì dùng `%LOCALAPPDATA%\claude-chat2md\logs`). Bấm **Mở thư mục log** để mở nhanh.

| File | Nội dung |
| --- | --- |
| `<ngày>.log` | Giống ô Nhật ký trên giao diện |
| `debug-<ngày>.log` | Chi tiết hơn, dùng khi cần báo lỗi: số dòng hỏng, phần nội dung chưa hỗ trợ, thời gian xử lý từng file, chi tiết lỗi |

Nhật ký chỉ ghi đường dẫn, số liệu và lỗi, **không chép nội dung chat**, nên có thể gửi kèm khi báo lỗi. File log tự xóa sau 7 ngày.

## Xử lý sự cố

| Thông báo / hiện tượng | Cách xử lý |
| --- | --- |
| "Không tìm thấy thư mục nguồn" | Kiểm tra lại ô thư mục dữ liệu chat. Nếu bạn dùng biến môi trường `CLAUDE_CONFIG_DIR`, dữ liệu nằm ở `<CLAUDE_CONFIG_DIR>\projects`. |
| "không có file .jsonl nào trong: …" | Thư mục đã chọn không chứa đoạn chat nào. Chọn thư mục `projects` hoặc một thư mục dự án bên trong nó. |
| Nhiều file bị "bỏ qua … file trống" | Bình thường: đó là các phiên mở ra nhưng chưa chat gì. |
| Có dòng "Lỗi: …" trong Nhật ký | Các file khác vẫn được chuyển. Bấm **Mở thư mục log**, gửi file `debug-<ngày>.log` cho người phát triển. |
| Nội dung trong file Markdown thiếu một phần | Nếu thiếu phần lệnh/kết quả hoặc suy luận: bật tùy chọn tương ứng rồi chuyển lại. Nếu thiếu tin nhắn: gửi file `debug-<ngày>.log`, trong đó có số dòng hỏng và loại nội dung chưa hỗ trợ của từng file. |

## Bản dòng lệnh (macOS/Linux)

```
claude-chat2md [-tools] [-thinking] [file.jsonl | thư-mục] [thư-mục-đầu-ra]
```

- `-tools`: kèm các lệnh Claude đã chạy và kết quả.
- `-thinking`: kèm phần suy luận.
- `-version`: in số phiên bản.
- Không ghi đường dẫn nguồn thì dùng `~/.claude/projects`; không ghi thư mục đầu ra thì dùng `claude-export` trong thư mục hiện tại.

Ví dụ: `claude-chat2md -tools ~/.claude/projects ~/Documents/claude-export`
