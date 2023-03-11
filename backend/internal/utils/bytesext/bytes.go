package bytesext

import (
	"crypto/rand"
	"io"
)

func Rand(size int) (bs []byte, err error) {
	bs = make([]byte, size)
	if _, err = io.ReadFull(rand.Reader, bs); err != nil {
		return
	}
	return
}
