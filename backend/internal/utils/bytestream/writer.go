package bytestream

import (
	"bytes"
	"encoding/binary"
)

type Writer struct {
	data *bytes.Buffer
}

func NewWriter() *Writer {
	return &Writer{
		data: bytes.NewBuffer(nil),
	}
}

func (w *Writer) ToBytes() []byte {
	return w.data.Bytes()
}

func (w *Writer) Bool(b bool) *Writer {
	if b {
		_ = w.data.WriteByte(byte(1))
	} else {
		_ = w.data.WriteByte(byte(0))
	}

	return w
}

func (w *Writer) Int(i int) *Writer {
	_ = binary.Write(w.data, binary.BigEndian, uint64(i))
	return w
}

func (w *Writer) Int32(i int32) *Writer {
	_ = binary.Write(w.data, binary.BigEndian, uint32(i))
	return w
}

func (w *Writer) Byte(b byte) *Writer {
	_ = w.data.WriteByte(b)
	return w
}

func (w *Writer) String(str string) *Writer {
	w.Int(len(str))
	_, _ = w.data.WriteString(str)
	return w
}
