package elfutil

import (
	"debug/elf"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
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
	// program headers detail
	phs := make([]map[string]any, 0, len(f.Progs))
	var hasTLSProg bool
	for _, p := range f.Progs {
		flags := ""
		if p.Flags&elf.PF_R != 0 {
			flags += "R"
		}
		if p.Flags&elf.PF_W != 0 {
			flags += "W"
		}
		if p.Flags&elf.PF_X != 0 {
			flags += "X"
		}
		if p.Type == elf.PT_TLS {
			hasTLSProg = true
		}
		phs = append(phs, map[string]any{
			"type":   p.Type.String(),
			"vaddr":  fmt.Sprintf("0x%x", p.Vaddr),
			"memsz":  p.Memsz,
			"filesz": p.Filesz,
			"flags":  flags,
			"align":  p.Align,
		})
	}
	m["program_headers_detail"] = phs
	var interp string
	var needed []string
	var rpath, runpath, buildID string
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			d, _ := io.ReadAll(p.Open())
			interp = strings.TrimRight(string(d), "\x00\n")
		}
	}
	// sections detail w/ flags & entropy (limited)
	sections := make([]map[string]any, 0, len(f.Sections))
	var textSize, rodataSize, dataSize, bssSize uint64
	var debugSections []string
	var hasSymtab bool
	var hasTLSSection bool
	var commentContent string
	for _, s := range f.Sections {
		ent := interface{}(nil)
		if (s.Name == ".text" || s.Name == ".rodata") && s.Size > 0 && s.Size < 4*1024*1024 { // cap for performance
			if b, e := s.Data(); e == nil {
				ent = fmt.Sprintf("%.4f", entropy(b))
			}
		}
		flags := sectionFlags(s.Flags)
		sections = append(sections, map[string]any{"name": s.Name, "size": s.Size, "type": s.Type.String(), "flags": flags, "entropy": ent})
		if s.Name == ".text" {
			textSize = s.Size
		}
		if s.Name == ".rodata" {
			rodataSize = s.Size
		}
		if s.Name == ".data" {
			dataSize = s.Size
		}
		if s.Name == ".bss" {
			bssSize = s.Size
		}
		if strings.HasPrefix(s.Name, ".debug") {
			debugSections = append(debugSections, s.Name)
		}
		if s.Type == elf.SHT_SYMTAB {
			hasSymtab = true
		}
		if s.Flags&elf.SHF_TLS != 0 {
			hasTLSSection = true
		}
		if s.Name == ".comment" {
			if b, e := s.Data(); e == nil {
				commentContent = strings.TrimSpace(string(trimNull(b)))
			}
		}
	}
	m["sections_detail"] = sections
	m["section_sizes"] = map[string]any{"text": textSize, "rodata": rodataSize, "data": dataSize, "bss": bssSize}
	// top sections by size (descending)
	top := make([]map[string]any, len(sections))
	copy(top, sections)
	sort.Slice(top, func(i, j int) bool { return top[i]["size"].(uint64) > top[j]["size"].(uint64) })
	if len(top) > 10 {
		top = top[:10]
	}
	m["top_sections"] = top
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
	// symbol tables
	var symCount, symExport, dynSymCount, dynSymExport int
	var exportedFuncs []string
	if syms, err := f.Symbols(); err == nil {
		symCount = len(syms)
		for _, s := range syms {
			if elf.ST_BIND(s.Info) == elf.STB_GLOBAL {
				symExport++
				if elf.ST_TYPE(s.Info) == elf.STT_FUNC {
					if len(exportedFuncs) < 50 {
						exportedFuncs = append(exportedFuncs, s.Name)
					}
				}
			}
		}
	}
	if dsyms, err := f.DynamicSymbols(); err == nil {
		dynSymCount = len(dsyms)
		for _, s := range dsyms {
			if elf.ST_BIND(s.Info) == elf.STB_GLOBAL {
				dynSymExport++
				if elf.ST_TYPE(s.Info) == elf.STT_FUNC {
					if len(exportedFuncs) < 50 {
						exportedFuncs = append(exportedFuncs, s.Name)
					}
				}
			}
		}
	}
	m["symbols"] = map[string]any{"sym_total": symCount, "sym_exported": symExport, "dyn_total": dynSymCount, "dyn_exported": dynSymExport, "exported_funcs_sample": exportedFuncs}
	// relocations count
	var relCount int
	for _, s := range f.Sections {
		if s.Type == elf.SHT_RELA || s.Type == elf.SHT_REL {
			if rels, e := s.Data(); e == nil { // rough count length/ entry guess
				if s.Type == elf.SHT_RELA && f.Class == elf.ELFCLASS64 {
					relCount += len(rels) / 24
				} else if s.Type == elf.SHT_REL && f.Class == elf.ELFCLASS64 {
					relCount += len(rels) / 16
				} else if s.Type == elf.SHT_RELA && f.Class == elf.ELFCLASS32 {
					relCount += len(rels) / 12
				} else if s.Type == elf.SHT_REL && f.Class == elf.ELFCLASS32 {
					relCount += len(rels) / 8
				}
			}
		}
	}
	m["relocations"] = map[string]any{"approx_total": relCount}
	// derive compiler from comment
	compiler := ""
	if commentContent != "" {
		lines := strings.Split(commentContent, "\n")
		for _, ln := range lines {
			l := strings.TrimSpace(ln)
			if l == "" {
				continue
			}
			compiler = l
			break
		}
	}
	// libc flavor
	libc := ""
	for _, n := range needed {
		ln := strings.ToLower(n)
		if strings.Contains(ln, "musl") {
			libc = "musl"
			break
		}
		if strings.Contains(ln, "glibc") || strings.Contains(ln, "libc.so") {
			libc = "glibc"
		}
	}
	// characteristics (improved)
	stripped := !hasSymtab // better heuristic
	static := (interp == "")
	pie := (f.Type == elf.ET_DYN)
	goBinary := false
	goBuildID := ""
	for _, s := range f.Sections {
		if s.Name == ".gopclntab" {
			goBinary = true
		}
		if s.Name == ".note.go.buildid" {
			if b, e := s.Data(); e == nil {
				goBuildID = string(trimNull(b))
			}
		}
	}
	hasTLS := hasTLSProg || hasTLSSection
	m["characteristics"] = map[string]any{
		"stripped":    stripped,
		"static":      static,
		"pie":         pie,
		"go_binary":   goBinary,
		"go_build_id": goBuildID,
		"tls":         hasTLS,
		"compiler":    compiler,
		"libc":        libc,
	}
	m["debug_info"] = map[string]any{"has": len(debugSections) > 0, "sections": debugSections}
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
func entropy(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	var freq [256]int
	for _, by := range b {
		freq[by]++
	}
	var e float64
	ln := float64(len(b))
	for _, c := range freq {
		if c == 0 {
			continue
		}
		p := float64(c) / ln
		e -= p * math.Log2(p)
	}
	return e
}
func sectionFlags(f elf.SectionFlag) string {
	var sb strings.Builder
	if f&elf.SHF_ALLOC != 0 {
		sb.WriteString("A")
	}
	if f&elf.SHF_EXECINSTR != 0 {
		sb.WriteString("X")
	}
	if f&elf.SHF_WRITE != 0 {
		sb.WriteString("W")
	}
	return sb.String()
}
func trimNull(b []byte) []byte {
	i := len(b)
	for i > 0 && (b[i-1] == 0 || b[i-1] == '\n') {
		i--
	}
	return b[:i]
}
