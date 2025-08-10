package elfutil

import (
	"debug/elf"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// AnalyzeBytes analyzes ELF file metadata from raw bytes (if ELF magic present)
func AnalyzeBytes(b []byte) (map[string]any, error) {
	if len(b) < 4 || b[0] != 0x7f || b[1] != 'E' || b[2] != 'L' || b[3] != 'F' {
		return nil, fmt.Errorf("not elf")
	}
	tmp, err := os.CreateTemp("", "elf-*\n")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return nil, err
	}
	tmp.Close()
	return AnalyzeFile(tmp.Name())
}

// AnalyzeFile opens an ELF file and extracts structured metadata.
func AnalyzeFile(path string) (map[string]any, error) {
	f, err := elf.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m := map[string]any{}
	m["class"] = f.Class.String()
	m["endianness"] = f.ByteOrder.String()
	m["type"] = f.Type.String()
	m["machine"] = f.Machine.String()
	m["entry"] = fmt.Sprintf("0x%x", f.Entry)
	m["osabi"] = f.OSABI.String()
	m["abi_version"] = f.ABIVersion
	m["sections"] = len(f.Sections)
	m["program_headers"] = len(f.Progs)
	var interp string
	var needed []string
	var rpath, runpath, buildID string
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			d, _ := io.ReadAll(p.Open())
			interp = strings.TrimRight(string(d), "\x00\n")
		}
	}
	for _, sec := range f.Sections {
		if sec.Type == elf.SHT_DYNAMIC {
			if dyn, _ := f.DynString(elf.DT_NEEDED); len(dyn) > 0 {
				needed = append(needed, dyn...)
			}
			if rs, err := f.DynString(elf.DT_RPATH); err == nil && len(rs) > 0 {
				rpath = strings.Join(rs, ":")
			}
			if rps, err := f.DynString(elf.DT_RUNPATH); err == nil && len(rps) > 0 {
				runpath = strings.Join(rps, ":")
			}
		}
		if sec.Type == elf.SHT_NOTE {
			if name, desc, _, err := readFirstNote(sec); err == nil {
				if name == "GNU" && len(desc) > 0 {
					buildID = fmt.Sprintf("%x", desc)
				}
			}
		}
	}
	if interp != "" {
		m["interp"] = interp
	}
	if len(needed) > 0 {
		m["needed"] = needed
	}
	if rpath != "" {
		m["rpath"] = rpath
	}
	if runpath != "" {
		m["runpath"] = runpath
	}
	if buildID != "" {
		m["build_id"] = buildID
	}
	sections := make([]map[string]any, 0, len(f.Sections))
	for _, s := range f.Sections {
		sections = append(sections, map[string]any{"name": s.Name, "size": s.Size, "type": s.Type.String()})
	}
	m["sections_detail"] = sections
	return m, nil
}

// TryAnalyzeBytes returns JSON string if ELF else nil.
func TryAnalyzeBytes(b []byte) *string {
	m, err := AnalyzeBytes(b)
	if err != nil {
		return nil
	}
	jb, _ := json.Marshal(m)
	s := string(jb)
	return &s
}

// readFirstNote minimal parsing of first note entry.
func readFirstNote(sec *elf.Section) (name string, desc []byte, typ uint32, err error) {
	data, err := sec.Data()
	if err != nil {
		return "", nil, 0, err
	}
	if len(data) < 12 {
		return "", nil, 0, fmt.Errorf("note too small")
	}
	nameSz := u32(data[0:4])
	descSz := u32(data[4:8])
	typ = u32(data[8:12])
	pos := 12
	if pos+int(nameSz) > len(data) {
		return "", nil, 0, fmt.Errorf("bad name size")
	}
	name = string(data[pos : pos+int(nameSz)])
	pos += align4(int(nameSz))
	if pos+int(descSz) > len(data) {
		return name, nil, typ, fmt.Errorf("bad desc size")
	}
	desc = data[pos : pos+int(descSz)]
	return
}

func u32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
func align4(v int) int {
	if v%4 == 0 {
		return v
	}
	return v + (4 - v%4)
}
