package file

import (
	"testing"
)

func TestMD5Sum(t *testing.T) {
	data := []byte("hello world")
	expected := "5eb63bbbe01eeed093cb22bb8f5acdc3" // precomputed
	if got := MD5Sum(data); got != expected {
		// helpful failure message
		t.Fatalf("md5 mismatch got=%s expected=%s", got, expected)
	}
}

func TestDetectMIME(t *testing.T) {
	cases := []struct {
		name   string
		data   []byte
		expect string
	}{
		{"plain", []byte("simple text content"), "text/plain; charset=utf-8"},
		{"json", []byte("{ \"k\": 1 }"), "application/json"},
	}
	for _, c := range cases {
		// potential variance for plain text detection between libraries: allow prefix match
		mime := DetectMIME(c.data, c.name)
		if c.name == "plain" {
			if len(mime) == 0 || mime[:10] != "text/plain" { // prefix check
				// If fails prefix check, that's a test failure
				t.Fatalf("plain text detection failed got=%s", mime)
			}
			continue
		}
		if mime != c.expect {
			t.Fatalf("%s detection mismatch got=%s expected=%s", c.name, mime, c.expect)
		}
	}
}
