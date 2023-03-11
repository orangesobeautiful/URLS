package strconvext

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

const testLen = 128

var longStr string
var logsBs []byte

func TestMain(m *testing.M) {
	longStr = strings.Repeat("a", testLen)
	logsBs = bytes.Repeat([]byte{'a'}, testLen)

	code := m.Run()
	os.Exit(code)
}

func TestB2S(t *testing.T) {
	convS := B2S(nil)
	if convS != "" {
		t.Errorf("B2S(nil) =%s, want empty string", convS)
	}

	convS = B2S([]byte{})
	if convS != string([]byte{}) {
		t.Errorf("B2S([]byte{}) =%s, want empty string", convS)
	}

	convS = B2S(logsBs)
	if convS != string(logsBs) {
		t.Errorf("B2S(logsBs) =%s, want %s", convS, string(logsBs))
	}
}

func TestS2B(t *testing.T) {
	convBs := S2B("")
	if !bytes.Equal(convBs, []byte("")) {
		t.Errorf("S2B(\"\") =%v, want %v", convBs, []byte(""))
	}

	convBs = S2B(longStr)
	if !bytes.Equal(convBs, []byte(longStr)) {
		t.Errorf("S2B(longStr) =%v, want %v", convBs, []byte(longStr))
	}
}

func BenchmarkB2SCopy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = string(logsBs)
	}
}

func BenchmarkB2S(b *testing.B) {
	for i := 0; i < b.N; i++ {
		B2S(logsBs)
	}
}

func BenchmarkS2BCopy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = []byte(longStr)
	}
}

func BenchmarkS2B(b *testing.B) {
	for i := 0; i < b.N; i++ {
		S2B(longStr)
	}
}
