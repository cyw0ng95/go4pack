package elfutil

import (
	"encoding/json"
	"os"
	"runtime"
	"testing"
)

// elfSamplePath returns a system ELF binary path instead of building one.
func elfSamplePath(t *testing.T) string {
	t.Helper()
	path := "/bin/uname"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("sample ELF %s not found: %v", path, err)
	}
	return path
}

func TestAnalyzeBytes_NotELF(t *testing.T) {
	if _, err := AnalyzeBytes([]byte("not elf")); err == nil {
		t.Fatalf("expected error for non-ELF bytes")
	}
}

func TestTryAnalyzeBytes_NotELF(t *testing.T) {
	if v := TryAnalyzeBytes([]byte("nope")); v != nil {
		t.Fatalf("expected nil for non-ELF TryAnalyzeBytes")
	}
}

func TestAnalyzeFile_ELFBinary(t *testing.T) {
	bin := elfSamplePath(t)
	info, err := AnalyzeFile(bin)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}
	keys := []string{"class", "endianness", "type", "machine", "entry", "sections", "program_headers", "characteristics"}
	for _, k := range keys {
		if _, ok := info[k]; !ok {
			t.Errorf("missing key %s", k)
		}
	}
	if chars, _ := info["characteristics"].(map[string]any); chars == nil {
		t.Fatalf("characteristics map missing or wrong type")
	}
}

func TestAnalyzeBytes_FullBinary(t *testing.T) {
	bin := elfSamplePath(t)
	b, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("read bin: %v", err)
	}
	info, err := AnalyzeBytes(b)
	if err != nil {
		t.Fatalf("AnalyzeBytes: %v", err)
	}
	if cls, ok := info["class"].(string); !ok || cls == "" {
		t.Errorf("class empty or missing")
	}
}

func TestSectionFlags(t *testing.T) {
	got := sectionFlags( /*A*/ 0x2 | /*X*/ 0x4 | /*W*/ 0x1)
	if got != "AXW" {
		t.Fatalf("unexpected flags order: %s", got)
	}
}

func TestEntropy(t *testing.T) {
	if e := entropy(make([]byte, 1024)); e != 0 {
		t.Errorf("entropy(zeros) expected 0 got %f", e)
	}
	uni := make([]byte, 0, 256*4)
	for i := 0; i < 4; i++ {
		for b := 0; b < 256; b++ {
			uni = append(uni, byte(b))
		}
	}
	if e := entropy(uni); e < 7.5 || e > 8.1 {
		t.Errorf("entropy(uniform) expected ~8 got %f", e)
	}
}

func TestTryAnalyzeBytes_ELF(t *testing.T) {
	bin := elfSamplePath(t)
	b, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("read bin: %v", err)
	}
	s := TryAnalyzeBytes(b)
	if s == nil {
		t.Fatalf("expected non-nil JSON string")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(*s), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["machine"]; !ok {
		t.Errorf("missing machine in JSON")
	}
}

func TestAnalyzeFile_PlatformStability(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Skip("cannot resolve self executable")
	}
	if runtime.GOOS == "windows" {
		t.Skip("windows exe not ELF")
	}
	info, err := AnalyzeFile(self)
	if err != nil {
		t.Skipf("self executable not ELF or unreadable: %v", err)
	}
	if info["sections"] == nil {
		t.Errorf("sections key missing in self analysis")
	}
}
