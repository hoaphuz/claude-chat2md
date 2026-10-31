// Command mkrsrc sinh file rsrc_windows_amd64.syso chứa icon ứng dụng,
// để Go linker nhúng vào .exe (icon trong Explorer, thanh tiêu đề, taskbar).
// Chỉ dùng thư viện chuẩn.
//
//	go run ./tools/mkrsrc [-ico assets/icons/png/app-icon.ico] [-o rsrc_windows_amd64.syso]
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
)

const (
	rtIcon      = 3
	rtGroupIcon = 14
	langEnUS    = 0x0409

	// iconGroupID là ID của RT_GROUP_ICON; main_windows.go nạp icon bằng ID này.
	iconGroupID = 1
)

func main() {
	ico := flag.String("ico", "assets/icons/png/app-icon.ico", "file .ico nguồn")
	out := flag.String("o", "rsrc_windows_amd64.syso", "file .syso đầu ra")
	flag.Parse()

	data, err := os.ReadFile(*ico)
	if err != nil {
		fail(err)
	}
	res, err := iconResources(data)
	if err != nil {
		fail(fmt.Errorf("%s: %w", *ico, err))
	}
	if err := os.WriteFile(*out, buildCOFF(res), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %s (%d icon images)\n", *out, len(res[rtIcon]))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "mkrsrc:", err)
	os.Exit(1)
}

// iconResources tách file .ico thành các RT_ICON (ID 1..n) và một RT_GROUP_ICON.
func iconResources(ico []byte) (map[uint16]map[uint16][]byte, error) {
	if len(ico) < 6 || binary.LittleEndian.Uint16(ico[2:]) != 1 {
		return nil, errors.New("not an .ico file")
	}
	n := int(binary.LittleEndian.Uint16(ico[4:]))
	if n == 0 || len(ico) < 6+16*n {
		return nil, errors.New("truncated icon directory")
	}
	icons := map[uint16][]byte{}
	var group bytes.Buffer
	group.Write(ico[:6]) // reserved, type=1, count
	for i := 0; i < n; i++ {
		e := ico[6+16*i : 6+16*(i+1)]
		size := binary.LittleEndian.Uint32(e[8:])
		off := binary.LittleEndian.Uint32(e[12:])
		if uint64(off)+uint64(size) > uint64(len(ico)) {
			return nil, fmt.Errorf("image %d out of range", i)
		}
		id := uint16(i + 1)
		icons[id] = ico[off : off+size]
		group.Write(e[:12]) // width..bytesInRes
		binary.Write(&group, binary.LittleEndian, id)
	}
	return map[uint16]map[uint16][]byte{
		rtIcon:      icons,
		rtGroupIcon: {iconGroupID: group.Bytes()},
	}, nil
}

func sortedKeys[V any](m map[uint16]V) []uint16 {
	keys := make([]uint16, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// buildCOFF tạo object COFF (AMD64) có một section .rsrc chứa cây tài nguyên
// type → id → ngôn ngữ, kèm relocation ADDR32NB cho từng data entry.
func buildCOFF(res map[uint16]map[uint16][]byte) []byte {
	type leaf struct {
		data      []byte
		entryOff  uint32 // vị trí IMAGE_RESOURCE_DATA_ENTRY
		dataOff   uint32 // vị trí dữ liệu
		langDirOf uint32 // vị trí thư mục ngôn ngữ
	}
	types := sortedKeys(res)
	dirSize := func(n int) uint32 { return 16 + 8*uint32(n) }

	// Bố cục: thư mục gốc, thư mục type, thư mục ngôn ngữ, data entry, dữ liệu.
	off := dirSize(len(types))
	typeDirOff := map[uint16]uint32{}
	for _, t := range types {
		typeDirOff[t] = off
		off += dirSize(len(res[t]))
	}
	var leaves []*leaf
	leafOf := map[[2]uint16]*leaf{}
	for _, t := range types {
		for _, id := range sortedKeys(res[t]) {
			l := &leaf{data: res[t][id], langDirOf: off}
			off += dirSize(1)
			leaves = append(leaves, l)
			leafOf[[2]uint16{t, id}] = l
		}
	}
	for _, l := range leaves {
		l.entryOff = off
		off += 16
	}
	for _, l := range leaves {
		off = (off + 7) &^ 7
		l.dataOff = off
		off += uint32(len(l.data))
	}
	sec := make([]byte, (off+7)&^7)

	le := binary.LittleEndian
	putDir := func(at uint32, n int) {
		le.PutUint16(sec[at+14:], uint16(n)) // NumberOfIdEntries
	}
	putEntry := func(dir uint32, i int, id uint16, target uint32, subdir bool) {
		at := dir + 16 + 8*uint32(i)
		le.PutUint32(sec[at:], uint32(id))
		if subdir {
			target |= 0x80000000
		}
		le.PutUint32(sec[at+4:], target)
	}

	putDir(0, len(types))
	for i, t := range types {
		putEntry(0, i, t, typeDirOff[t], true)
		ids := sortedKeys(res[t])
		putDir(typeDirOff[t], len(ids))
		for j, id := range ids {
			l := leafOf[[2]uint16{t, id}]
			putEntry(typeDirOff[t], j, id, l.langDirOf, true)
			putDir(l.langDirOf, 1)
			putEntry(l.langDirOf, 0, langEnUS, l.entryOff, false)
		}
	}
	for _, l := range leaves {
		le.PutUint32(sec[l.entryOff:], l.dataOff) // RVA, được relocation sửa lại
		le.PutUint32(sec[l.entryOff+4:], uint32(len(l.data)))
		copy(sec[l.dataOff:], l.data)
	}

	const fileHdr, secHdr = 20, 40
	rawOff := uint32(fileHdr + secHdr)
	relocOff := rawOff + uint32(len(sec))
	symOff := relocOff + 10*uint32(len(leaves))

	var b bytes.Buffer
	w := func(v any) { binary.Write(&b, le, v) }
	// IMAGE_FILE_HEADER
	w(uint16(0x8664)) // AMD64
	w(uint16(1))      // NumberOfSections
	w(uint32(0))      // TimeDateStamp
	w(symOff)         // PointerToSymbolTable
	w(uint32(1))      // NumberOfSymbols
	w(uint16(0))      // SizeOfOptionalHeader
	w(uint16(0))      // Characteristics
	// IMAGE_SECTION_HEADER
	b.WriteString(".rsrc\x00\x00\x00")
	w(uint32(0))           // VirtualSize
	w(uint32(0))           // VirtualAddress
	w(uint32(len(sec)))    // SizeOfRawData
	w(rawOff)              // PointerToRawData
	w(relocOff)            // PointerToRelocations
	w(uint32(0))           // PointerToLinenumbers
	w(uint16(len(leaves))) // NumberOfRelocations
	w(uint16(0))           // NumberOfLinenumbers
	w(uint32(0x40000040))  // INITIALIZED_DATA | MEM_READ
	b.Write(sec)
	for _, l := range leaves {
		w(l.entryOff) // VirtualAddress
		w(uint32(0))  // SymbolTableIndex
		w(uint16(3))  // IMAGE_REL_AMD64_ADDR32NB
	}
	// Symbol .rsrc (section symbol)
	b.WriteString(".rsrc\x00\x00\x00")
	w(uint32(0)) // Value
	w(int16(1))  // SectionNumber
	w(uint16(0)) // Type
	w(uint8(3))  // IMAGE_SYM_CLASS_STATIC
	w(uint8(0))  // NumberOfAuxSymbols
	// String table rỗng
	w(uint32(4))
	return b.Bytes()
}
