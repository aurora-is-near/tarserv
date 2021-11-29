package tarindex

import (
	"os"
)

func isRegular(fi os.FileInfo) bool {
	mode := fi.Mode()
	return mode & ^os.ModeType == mode
}

func isLink(fi os.FileInfo) bool {
	mode := fi.Mode()
	return mode&os.ModeSymlink != 0
}
