package storage

import (
	"errors"
	"os"
	"syscall"
)

func NewRequestDir() (string, error) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs("/dev/shm", &fs); err != nil || uint64(fs.Type) != 0x01021994 {
		return "", errors.New("volatile temporary storage unavailable")
	}
	return os.MkdirTemp("/dev/shm", "file-converter-")
}
