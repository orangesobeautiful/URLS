package bytestream

import (
	"testing"
)

func TestWriterAndReader(t *testing.T) {
	wBool := true
	wByte := byte(123)
	wInt := 456
	var wInt32 int32 = 789
	wStr := "a test message"

	w := NewWriter()
	bs := w.Bool(wBool).Byte(wByte).Int(wInt).Int32(wInt32).String(wStr).ToBytes()

	r := NewReader(bs)

	var rBool bool
	var rByte byte
	var rInt int
	var rInt32 int32
	var rStr string

	r.Bool(&rBool)
	if r.HasErr() {
		t.Fatalf("read bool has error")
	}
	if rBool != wBool {
		t.Fatalf("rBool = %t, want %t", rBool, wBool)
	}

	r.Byte(&rByte)
	if r.HasErr() {
		t.Fatalf("read byte has error")
	}
	if rByte != wByte {
		t.Fatalf("rByte = %d, want %d", rByte, wByte)
	}

	r.Int(&rInt)
	if r.HasErr() {
		t.Fatalf("read int has error")
	}
	if rInt != wInt {
		t.Fatalf("rInt = %d, want %d", rInt, wInt)
	}

	r.Int32(&rInt32)
	if r.HasErr() {
		t.Fatalf("read int32 has error")
	}
	if rInt32 != wInt32 {
		t.Fatalf("rInt32 = %d, want %d", rInt32, wInt32)
	}

	r.String(&rStr)
	if r.HasErr() {
		t.Fatalf("read string has error")
	}
	if rStr != wStr {
		t.Fatalf("rStr = \"%s\", want \"%s\"", rStr, wStr)
	}

	var tmp byte
	r.Byte(&tmp)
	if !r.HasErr() {
		t.Fatalf("read to eof but has no error")
	}
}
