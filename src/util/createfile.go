package util

import "os"

func CreateFile(filename string) (*os.File, error) {
	return os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0640)
}
