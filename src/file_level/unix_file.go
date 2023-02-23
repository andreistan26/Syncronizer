package file_level

import (
	"crypto/md5"
	"io"
	"os"
)

func GetFileMD5(filename string) ([16]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return [16]byte{}, err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return [16]byte{}, err
	}
	return [16]byte(h.Sum(nil)), err
}
