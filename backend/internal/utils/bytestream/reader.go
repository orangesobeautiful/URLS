package bytestream

import (
	"encoding/binary"
)

type Reader struct {
	data    []byte
	dataLen int
	cur     int
	hasErr  bool
}

func NewReader(data []byte) *Reader {
	return &Reader{
		data:    data,
		dataLen: len(data),
		cur:     0,
	}
}

func (r *Reader) HasErr() bool {
	return r.hasErr
}

func (r *Reader) Bool(b *bool) *Reader {
	if r.cur+1-1 >= r.dataLen {
		r.hasErr = true
		return r
	}

	if r.data[r.cur] == 0 {
		*b = false
	} else {
		*b = true
	}
	r.cur++
	return r
}

func (r *Reader) Int(i *int) *Reader {
	if r.cur+8-1 >= r.dataLen {
		r.hasErr = true
		return r
	}

	res := binary.BigEndian.Uint64(r.data[r.cur : r.cur+8])
	*i = int(res)
	r.cur += 8
	return r
}

func (r *Reader) Int32(i *int32) *Reader {
	if r.cur+4-1 >= r.dataLen {
		r.hasErr = true
		return r
	}

	res := binary.BigEndian.Uint32(r.data[r.cur : r.cur+4])
	*i = int32(res)
	r.cur += 4
	return r
}

func (r *Reader) Byte(b *byte) *Reader {
	if r.cur+1-1 >= r.dataLen {
		r.hasErr = true
		return r
	}

	*b = r.data[r.cur]
	r.cur++
	return r
}

func (r *Reader) String(str *string) *Reader {
	var orgCur = r.cur
	var strLen = -1
	r.Int(&strLen)
	if strLen < 0 {
		r.hasErr = true
		return r
	}

	if r.cur+strLen-1 >= r.dataLen {
		r.cur = orgCur
		r.hasErr = true
		return r
	}

	*str = string(r.data[r.cur : r.cur+strLen])
	r.cur += strLen
	return r
}
