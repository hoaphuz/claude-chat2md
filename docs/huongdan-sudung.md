# Hướng dẫn sử dụng claude-chat2md

## Giới thiệu

claude-chat2md chuyển các đoạn chat với Claude Code (trong VS Code hoặc terminal) thành file Markdown để đọc lại, lưu trữ, tìm kiếm hoặc chia sẻ. Mỗi đoạn chat thành một file `.md`, có tiêu đề, thời gian và lượt nói của bạn và Claude.

Nên dùng khi:

- Muốn lưu lại cách đã giải quyết một vấn đề cùng Claude để xem lại sau.
- Sắp đổi tên hoặc chuyển thư mục dự án và muốn giữ một bản chat dễ đọc.
- Cần gửi nội dung một đoạn chat cho người khác xem.

Công cụ **chỉ đọc** dữ liệu chat của Claude Code, không sửa hay xóa gì trong đó.

Yêu cầu máy: Windows 10 hoặc 11 bản 64-bit. Không cần cài thêm gì. Trên macOS/Linux có bản dòng lệnh (xem cuối tài liệu).

Tài liệu này viết cho phiên bản 0.7.0. Số phiên bản hiện trên tiêu đề cửa sổ.

## Cài đặt và mở công cụ

Công cụ là bản portable, không cần cài đặt.

1. Giải nén file `claude-chat2md-portable-<phiên bản>.zip` vào một thư mục bạn có quyền ghi, ví dụ `D:\Tools\claude-chat2md`.
2. Chạy `claude-chat2md.exe`.
3. Nếu Windows hiện cảnh báo "Windows protected your PC" (file exe chưa ký số), bấm **More info** → **Run anyway**.

## Chuyển chat sang Markdown

1. **Dữ liệu chat của Claude Code.** Mặc định là `%USERPROFILE%\.claude\projects`, nơi Claude Code lưu mọi đoạn chat. Thường không cần đổi. Công cụ nhớ nguồn dùng lần cuối; bấm nút sổ xuống ở cuối ô để chọn lại một trong 10 nguồn dùng gần đây (thư mục hoặc danh sách tệp). Có thể chọn:
    - Cả thư mục `projects` (nút **Thư mục…**): chuyển toàn bộ chat của mọi dự án.
    - Một thư mục con bên trong (nút **Thư mục…**): chỉ chuyển chat của một dự án (xem mục *Tìm chat của một dự án*).
    - Một hoặc nhiều file chat (nút **Chọn tệp…**): chỉ chuyển đúng các đoạn chat đó. Xem mục *Chuyển riêng một vài đoạn chat*.
2. **Thư mục lưu tệp Markdown.** Lần đầu là `Documents\claude-export`. Bấm **Chọn…** để đổi, hoặc bấm nút sổ xuống ở cuối ô để chọn lại một trong 10 thư mục dùng gần đây. Cả hai ô đều được nhớ, lần mở sau vẫn giữ nguyên (lưu ở file `settings.json` cạnh `claude-chat2md.exe`; nếu thư mục đó không ghi được thì ở `%APPDATA%\claude-chat2md`).
3. **Chọn tùy chọn** (xem bảng dưới).
4. **Bấm "Chuyển đổi"**. Ô Nhật ký hiện từng file đã xuất và dòng tổng kết, ví dụ `Xong: 3 file Markdown mới/cập nhật, 60 không đổi, bỏ qua 0 file trống, 0 lỗi.`
5. Nếu đã đánh dấu "Mở thư mục đầu ra khi xong", thư mục kết quả tự mở. Lúc khác có thể bấm **Mở thư mục đầu ra**.

| Tùy chọn | Tác dụng |
| --- | --- |
| Kèm các lệnh Claude đã chạy và kết quả | Thêm các lần Claude đọc file, chạy lệnh, sửa code… cùng kết quả. Mỗi mục thu gọn, bấm để mở. Kết quả quá 3000 ký tự được cắt bớt. Bỏ chọn thì file chỉ còn lời trao đổi, ngắn và dễ đọc hơn. |
| Kèm phần suy luận (thinking) của Claude | Thêm phần Claude suy nghĩ trước khi trả lời (nếu có), cũng thu gọn. |
| Đặt tên tệp theo tên phiên chat | Đặt tên file `.md` theo tên phiên (tên bạn tự đặt, hoặc tên Claude Code tự đặt cho phiên) thay vì mã phiên dạng `c258cffe-…`. Ký tự không hợp lệ trong tên file bị thay bằng dấu cách, tên dài quá 100 ký tự bị cắt. Hai phiên trùng tên trong cùng thư mục thì file sau có thêm `(2)`, `(3)`… Mặc định bật. |
| Chỉ chuyển chat mới hoặc đã thay đổi | Bỏ qua các đoạn chat đã xuất và không đổi gì kể từ lần trước; chỉ chuyển chat mới, chat có thêm tin nhắn, hoặc chat có file `.md` đã bị xóa. Đổi tùy chọn nội dung (lệnh, suy luận, cách đặt tên) hoặc cập nhật lên phiên bản mới thì mọi chat được chuyển lại một lần. Bỏ chọn để luôn chuyển lại toàn bộ. Mặc định bật. Xem mục *Chỉ chuyển phần thay đổi*. |
| Mở thư mục đầu ra khi xong | Tự mở thư mục kết quả sau khi chuyển xong. Mặc định bật. |

## Chuyển riêng một vài đoạn chat

1. Bấm **Chọn tệp…** cạnh ô dữ liệu chat. Hộp thoại mở sẵn thư mục đang ghi trong ô (hoặc thư mục `projects`).
2. Vào thư mục của dự án (xem mục *Tìm chat của một dự án*), chọn file `.jsonl`. Giữ **Ctrl** để chọn từng file, hoặc **Shift** để chọn một dãy liên tiếp. Mẹo: chuyển hộp thoại sang chế độ xem **Details** và sắp xếp theo **Date modified** để thấy ngay các đoạn chat gần đây.
3. Bấm **Open**. Ô dữ liệu chat hiện danh sách file đã chọn, cách nhau bằng dấu `;`. Bạn cũng có thể tự gõ hoặc dán nhiều đường dẫn theo cách này.
4. Bấm **Chuyển đổi**.

Cần biết:

- File chọn riêng **luôn được chuyển**, kể cả khi đang bật "Chỉ chuyển chat mới hoặc đã thay đổi" và đoạn chat chưa đổi gì.
- File nằm trong thư mục `projects` được đặt vào thư mục con theo dự án, giống hệt khi chuyển cả thư mục `projects`. Nhờ vậy, chọn lại vài đoạn chat sẽ cập nhật đúng file đã xuất trước đó, không tạo bản trùng. File nằm ở nơi khác được đặt thẳng vào thư mục đầu ra.
- Muốn quay lại chuyển cả thư mục, bấm **Thư mục…** và chọn thư mục.

## File Markdown nhận được

Thư mục đầu ra giữ nguyên cấu trúc của thư mục nguồn: mỗi dự án một thư mục con, mỗi đoạn chat một file đặt theo tên phiên (hoặc theo mã phiên nếu bỏ chọn "Đặt tên tệp theo tên phiên chat"), ví dụ:

```
claude-export\
  d--Projects-my-app\
    Sửa lỗi đăng nhập.md                      (theo tên phiên)
    c258cffe-93d4-4b44-80c5-31ba05815f96.md   (theo mã phiên)
```

Mỗi file gồm:

- **Tiêu đề**: tên bạn đã đặt cho đoạn chat; nếu không có thì lấy tên Claude Code tự đặt cho phiên, tóm tắt của Claude Code, hoặc dòng đầu tiên bạn gõ. Tên file (khi bật tùy chọn) lấy đúng tiêu đề này.
- **Session** (mã phiên), **Thư mục dự án** và **Bắt đầu** (giờ máy bạn). Mã phiên luôn được ghi, kể cả khi tên file đặt theo tên phiên (xem mục *Tra cứu bằng mã phiên*).
- Các lượt **👤 Bạn** và **🤖 Claude**, mỗi lượt có giờ gửi. Nhiều tin liên tiếp của cùng một bên được gộp dưới một tiêu đề.

Những gì **không** có trong file:

- Các đoạn hệ thống tự chèn vào tin nhắn (nhắc nhở, nội dung lệnh `/` bên trong). Lệnh `/` bạn gõ vẫn giữ tên lệnh, ví dụ `/init`.
- Hội thoại của các agent phụ (subagent) do Claude tự tạo.
- Hình ảnh: chỉ hiện chữ *[hình ảnh đính kèm]*.

Đoạn chat không có tin nhắn nào sẽ bị bỏ qua và được tính vào "bỏ qua … file trống" (từ lần chuyển sau, nếu vẫn không đổi, nó được tính vào "không đổi").

Xem file đẹp nhất bằng VS Code (Ctrl+Shift+V để xem trước) hoặc bất kỳ trình đọc Markdown nào.

## Chỉ chuyển phần thay đổi

Khi bật "Chỉ chuyển chat mới hoặc đã thay đổi", công cụ ghi lại những gì đã xuất vào file `.claude-chat2md-state.json` trong thư mục đầu ra. Lần sau, nó so kích thước và thời điểm sửa của từng file chat gốc với lần trước để biết chat nào cần chuyển. Nhờ vậy có thể bấm "Chuyển đổi" thường xuyên mà chỉ mất vài giây.

- Đừng xóa hay sửa file `.claude-chat2md-state.json`. Nếu lỡ xóa, lần sau công cụ chuyển lại toàn bộ rồi tạo lại file này, không mất dữ liệu.
- Mỗi thư mục đầu ra có file trạng thái riêng. Đổi sang thư mục đầu ra khác thì lần đầu sẽ chuyển toàn bộ.
- Chat đang diễn ra sẽ được chuyển lại mỗi lần bấm, vì file gốc của nó liên tục thay đổi.

## Tra cứu bằng mã phiên

Đầu mỗi file Markdown có dòng mã phiên, ví dụ:

```
- **Session:** `c258cffe-93d4-4b44-80c5-31ba05815f96`
```

Dùng mã này để:

- **Tìm file Markdown của một phiên** khi tên file là tên phiên: mở thư mục đầu ra trong VS Code rồi tìm mã bằng Ctrl+Shift+F, hoặc chạy trong PowerShell:
  ```powershell
  Select-String -Path "$env:USERPROFILE\Documents\claude-export\*\*.md" -Pattern "c258cffe" -List
  ```
  Chỉ cần gõ vài ký tự đầu của mã.
- **Tìm lại file chat gốc**: file gốc tên là `<mã phiên>.jsonl`, nằm trong `.claude\projects\<thư mục cùng tên với thư mục con chứa file .md>`.
- **Mở lại phiên đó trong Claude Code** để chat tiếp: chạy `claude --resume <mã phiên>` trong thư mục dự án.

## Tìm chat của một dự án

Claude Code đặt tên thư mục chat theo đường dẫn dự án, thay các ký tự `:`, `\`, dấu cách, dấu chấm… bằng dấu `-`. Ví dụ dự án `D:\Projects\my-app` có chat nằm trong:

```
%USERPROFILE%\.claude\projects\d--Projects-my-app
```

Chọn đúng thư mục này làm thư mục nguồn để chỉ xuất chat của dự án đó.

**Khi đổi tên hoặc chuyển thư mục dự án**, Claude Code sẽ không còn hiện các đoạn chat cũ của dự án (dữ liệu vẫn còn, chỉ nằm ở thư mục mang tên cũ). Nên xuất ra Markdown trước khi đổi tên. Nếu muốn Claude Code vẫn thấy lịch sử, hãy đóng VS Code rồi đổi tên luôn thư mục chat trong `.claude\projects` cho khớp với đường dẫn mới.

## Lưu ý quan trọng

- **Chuyển lại sẽ ghi đè.** Khi một đoạn chat được chuyển lại (vì có tin nhắn mới, hoặc đã bỏ chọn "Chỉ chuyển chat mới hoặc đã thay đổi"), file `.md` của nó bị ghi đè. Nếu đã tự sửa hoặc ghi chú vào file Markdown, hãy chép sang chỗ khác hoặc chọn thư mục đầu ra khác.
- **Khi tên phiên đổi** (bạn đổi tên, Claude Code đặt lại tên, hoặc bật/tắt "Đặt tên tệp theo tên phiên chat"), lần chuyển sau tạo file mang tên mới và tự xóa file tên cũ, nhưng chỉ khi file cũ chưa bị bạn sửa. File cũ đã sửa được giữ lại, Nhật ký ghi "Giữ lại file cũ vì đã bị sửa".
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
| "Không tìm thấy nguồn" | Kiểm tra lại ô dữ liệu chat (đường dẫn gõ sai, hoặc file đã bị xóa/đổi tên). Nhiều đường dẫn phải cách nhau bằng dấu `;`. Nếu bạn dùng biến môi trường `CLAUDE_CONFIG_DIR`, dữ liệu nằm ở `<CLAUDE_CONFIG_DIR>\projects`. |
| "không có file .jsonl nào trong: …" | Thư mục đã chọn không chứa đoạn chat nào. Chọn thư mục `projects` hoặc một thư mục dự án bên trong nó. |
| Nhiều file bị "bỏ qua … file trống" | Bình thường: đó là các phiên mở ra nhưng chưa chat gì. |
| Có dòng "Lỗi: …" trong Nhật ký | Các file khác vẫn được chuyển. Bấm **Mở thư mục log**, gửi file `debug-<ngày>.log` cho người phát triển. |
| Nội dung trong file Markdown thiếu một phần | Nếu thiếu phần lệnh/kết quả hoặc suy luận: bật tùy chọn tương ứng rồi chuyển lại. Nếu thiếu tin nhắn: gửi file `debug-<ngày>.log`, trong đó có số dòng hỏng và loại nội dung chưa hỗ trợ của từng file. |

## Bản dòng lệnh (macOS/Linux)

```
claude-chat2md [-tools] [-thinking] [-title] [-update] [file.jsonl | thư-mục] [thư-mục-đầu-ra]
claude-chat2md [cờ…] -o thư-mục-đầu-ra file.jsonl|thư-mục …
```

- `-tools`: kèm các lệnh Claude đã chạy và kết quả.
- `-thinking`: kèm phần suy luận.
- `-title`: đặt tên file theo tên phiên chat (mặc định tắt ở bản dòng lệnh: đặt theo mã phiên).
- `-update`: chỉ chuyển chat mới hoặc đã thay đổi kể từ lần xuất trước (mặc định tắt ở bản dòng lệnh).
- `-o <thư-mục>`: thư mục đầu ra; khi dùng cờ này, mọi tham số còn lại là nguồn, nên chuyển được nhiều file cùng lúc.
- `-version`: in số phiên bản.
- Không ghi đường dẫn nguồn thì dùng `~/.claude/projects`; không ghi thư mục đầu ra thì dùng `claude-export` trong thư mục hiện tại.

Ví dụ:

- Cả thư mục: `claude-chat2md -tools ~/.claude/projects ~/Documents/claude-export`
- Vài file: `claude-chat2md -title -o ~/Documents/claude-export ~/.claude/projects/my-app/a.jsonl ~/.claude/projects/my-app/b.jsonl`
