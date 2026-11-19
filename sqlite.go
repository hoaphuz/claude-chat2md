package main

// Bộ đọc SQLite tối thiểu, chỉ đọc, chỉ thư viện chuẩn: đủ để duyệt một bảng rowid
// (vd. ItemTable trong state.vscdb của VS Code). Không hỗ trợ SQL, chỉ mục hay ghi.
// Định dạng: https://www.sqlite.org/fileformat2.html

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
)

const (
	sqliteMaxFile = 512 << 20 // không đọc file lớn hơn (state.vscdb thường vài MB)
	sqliteMaxRec  = 64 << 20  // giới hạn kích thước một dòng
)

var errSQLiteFormat = errors.New("không phải file SQLite hợp lệ")

type sqliteDB struct {
	data     []byte
	pageSize int
	usable   int
	nPages   uint32
	wal      map[uint32][]byte // trang mới nhất trong file -wal (nếu có)
}

// openSQLite đọc toàn bộ file (và file -wal nếu database ở chế độ WAL).
func openSQLite(path string) (*sqliteDB, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.Size() > sqliteMaxFile {
		return nil, fmt.Errorf("file quá lớn (%d byte)", fi.Size())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 100 || string(data[:16]) != "SQLite format 3\x00" {
		return nil, errSQLiteFormat
	}
	ps := int(binary.BigEndian.Uint16(data[16:18]))
	if ps == 1 {
		ps = 65536
	}
	if ps < 512 || ps&(ps-1) != 0 {
		return nil, errSQLiteFormat
	}
	db := &sqliteDB{data: data, pageSize: ps, usable: ps - int(data[20])}
	if db.usable < 480 {
		return nil, errSQLiteFormat
	}
	if enc := binary.BigEndian.Uint32(data[56:60]); enc > 1 {
		return nil, errors.New("chỉ hỗ trợ SQLite mã hóa UTF-8")
	}
	db.nPages = uint32(len(data) / ps)
	if data[18] == 2 || data[19] == 2 { // chế độ WAL
		if w, err := os.ReadFile(path + "-wal"); err == nil {
			db.applyWAL(w)
		}
	}
	return db, nil
}

// applyWAL lấy các trang đã commit trong file -wal (bỏ khung của thế hệ cũ, khác salt).
func (db *sqliteDB) applyWAL(w []byte) {
	if len(w) < 32 {
		return
	}
	if m := binary.BigEndian.Uint32(w[0:4]); m != 0x377f0682 && m != 0x377f0683 {
		return
	}
	if int(binary.BigEndian.Uint32(w[8:12])) != db.pageSize {
		return
	}
	salt1, salt2 := binary.BigEndian.Uint32(w[16:20]), binary.BigEndian.Uint32(w[20:24])
	pending := map[uint32][]byte{}
	committed := map[uint32][]byte{}
	nPages := db.nPages
	for off := 32; off+24+db.pageSize <= len(w); off += 24 + db.pageSize {
		h := w[off : off+24]
		if binary.BigEndian.Uint32(h[8:12]) != salt1 || binary.BigEndian.Uint32(h[12:16]) != salt2 {
			break
		}
		pg := binary.BigEndian.Uint32(h[0:4])
		pending[pg] = w[off+24 : off+24+db.pageSize]
		if size := binary.BigEndian.Uint32(h[4:8]); size != 0 { // khung commit
			for k, v := range pending {
				committed[k] = v
			}
			pending = map[uint32][]byte{}
			nPages = size
		}
	}
	if len(committed) > 0 {
		db.wal = committed
		db.nPages = nPages
	}
}

func (db *sqliteDB) page(n uint32) ([]byte, error) {
	if n == 0 || n > db.nPages {
		return nil, fmt.Errorf("trang %d ngoài phạm vi", n)
	}
	if p, ok := db.wal[n]; ok {
		return p, nil
	}
	off := int(n-1) * db.pageSize
	if off+db.pageSize > len(db.data) {
		return nil, fmt.Errorf("trang %d ngoài phạm vi", n)
	}
	return db.data[off : off+db.pageSize], nil
}

// varint đọc số nguyên độ dài thay đổi của SQLite (1–9 byte).
func sqliteVarint(b []byte) (uint64, int) {
	var v uint64
	for i := 0; i < 8; i++ {
		if i >= len(b) {
			return 0, 0
		}
		v = v<<7 | uint64(b[i]&0x7f)
		if b[i] < 0x80 {
			return v, i + 1
		}
	}
	if len(b) < 9 {
		return 0, 0
	}
	return v<<8 | uint64(b[8]), 9
}

// scanTable duyệt mọi dòng của bảng có trang gốc root, gọi fn với các cột đã giải mã
// (nil, int64, float64, string hoặc []byte).
func (db *sqliteDB) scanTable(root uint32, fn func(cols []any) error) error {
	visited := map[uint32]bool{}
	var walk func(pg uint32, depth int) error
	walk = func(pg uint32, depth int) error {
		if depth > 64 || visited[pg] {
			return errors.New("cây B-tree bị vòng lặp")
		}
		visited[pg] = true
		p, err := db.page(pg)
		if err != nil {
			return err
		}
		hdr := 0
		if pg == 1 {
			hdr = 100
		}
		if hdr+8 > len(p) {
			return errSQLiteFormat
		}
		kind := p[hdr]
		nCells := int(binary.BigEndian.Uint16(p[hdr+3 : hdr+5]))
		ptrs := hdr + 8
		if kind == 0x05 {
			ptrs = hdr + 12
		}
		if ptrs+2*nCells > len(p) {
			return errSQLiteFormat
		}
		for i := 0; i < nCells; i++ {
			off := int(binary.BigEndian.Uint16(p[ptrs+2*i:]))
			if off >= db.usable {
				return errSQLiteFormat
			}
			switch kind {
			case 0x05: // trang trong (interior) của bảng: con trái + rowid
				if off+4 > len(p) {
					return errSQLiteFormat
				}
				if err := walk(binary.BigEndian.Uint32(p[off:]), depth+1); err != nil {
					return err
				}
			case 0x0D: // trang lá của bảng
				payload, err := db.cellPayload(p, off)
				if err != nil {
					return err
				}
				cols, err := decodeRecord(payload)
				if err != nil {
					return err
				}
				if err := fn(cols); err != nil {
					return err
				}
			default:
				return fmt.Errorf("trang %d không phải trang bảng (loại 0x%02x)", pg, kind)
			}
		}
		if kind == 0x05 {
			return walk(binary.BigEndian.Uint32(p[hdr+8:]), depth+1)
		}
		return nil
	}
	return walk(root, 0)
}

// cellPayload trả về toàn bộ payload của một ô trên trang lá, nối cả các trang tràn.
func (db *sqliteDB) cellPayload(p []byte, off int) ([]byte, error) {
	size, n := sqliteVarint(p[off:db.usable])
	if n == 0 || size > sqliteMaxRec {
		return nil, errSQLiteFormat
	}
	off += n
	if _, n = sqliteVarint(p[off:db.usable]); n == 0 { // rowid
		return nil, errSQLiteFormat
	}
	off += n
	total := int(size)
	u := db.usable
	maxLocal := u - 35
	local := total
	if total > maxLocal {
		minLocal := (u-12)*32/255 - 23
		local = minLocal + (total-minLocal)%(u-4)
		if local > maxLocal {
			local = minLocal
		}
	}
	if off+local > u || (local < total && off+local+4 > u) {
		return nil, errSQLiteFormat
	}
	out := make([]byte, 0, total)
	out = append(out, p[off:off+local]...)
	if local == total {
		return out, nil
	}
	next := binary.BigEndian.Uint32(p[off+local:])
	for seen := 0; len(out) < total; seen++ {
		if next == 0 || seen > int(db.nPages) {
			return nil, errors.New("chuỗi trang tràn bị hỏng")
		}
		op, err := db.page(next)
		if err != nil {
			return nil, err
		}
		chunk := op[4:u]
		if rest := total - len(out); len(chunk) > rest {
			chunk = chunk[:rest]
		}
		out = append(out, chunk...)
		next = binary.BigEndian.Uint32(op[0:4])
	}
	return out, nil
}

// decodeRecord giải mã một bản ghi (record format) thành danh sách cột.
func decodeRecord(b []byte) ([]any, error) {
	hlen, n := sqliteVarint(b)
	if n == 0 || hlen > uint64(len(b)) || int(hlen) < n {
		return nil, errSQLiteFormat
	}
	var types []uint64
	for pos := n; pos < int(hlen); {
		t, k := sqliteVarint(b[pos:hlen])
		if k == 0 {
			return nil, errSQLiteFormat
		}
		types = append(types, t)
		pos += k
	}
	cols := make([]any, 0, len(types))
	body := b[hlen:]
	for _, t := range types {
		var size int
		switch {
		case t == 0 || t == 8 || t == 9:
			size = 0
		case t >= 1 && t <= 4:
			size = int(t)
		case t == 5:
			size = 6
		case t == 6 || t == 7:
			size = 8
		case t >= 12:
			size = int((t - 12) / 2)
		default:
			return nil, errSQLiteFormat
		}
		if size > len(body) {
			return nil, errSQLiteFormat
		}
		v := body[:size]
		body = body[size:]
		switch {
		case t == 0:
			cols = append(cols, nil)
		case t == 8:
			cols = append(cols, int64(0))
		case t == 9:
			cols = append(cols, int64(1))
		case t <= 6:
			x := int64(int8(v[0])) // có dấu, big-endian
			for _, c := range v[1:] {
				x = x<<8 | int64(c)
			}
			cols = append(cols, x)
		case t == 7:
			cols = append(cols, math.Float64frombits(binary.BigEndian.Uint64(v)))
		case t%2 == 0:
			cols = append(cols, append([]byte(nil), v...))
		default:
			cols = append(cols, string(v))
		}
	}
	return cols, nil
}

// tableRoot tìm trang gốc của bảng trong sqlite_schema (trang 1).
func (db *sqliteDB) tableRoot(name string) (uint32, error) {
	var root uint32
	err := db.scanTable(1, func(c []any) error {
		if len(c) >= 4 && c[0] == "table" {
			if n, _ := c[1].(string); strings.EqualFold(n, name) {
				if r, ok := c[3].(int64); ok && r > 0 {
					root = uint32(r)
				}
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if root == 0 {
		return 0, fmt.Errorf("không có bảng %s", name)
	}
	return root, nil
}

// sqliteText trả về giá trị cột dạng chữ (TEXT hoặc BLOB), "" nếu khác kiểu.
func sqliteText(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	}
	return ""
}
