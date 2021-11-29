package main

import (
	"fmt"
	"os"
	"path"

	"github.com/aurora-is/tarserv/src/util"

	"github.com/aurora-is/tarserv/src/tarindex"
)

func main() {
	if len(os.Args) != 3 {
		_, _ = fmt.Fprintf(os.Stderr, "%s <indexfile> <source directory>\n", path.Base(os.Args[0]))
		os.Exit(1)
	}
	f, err := util.CreateFile(os.Args[1])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: Error opening index file: %s\n", path.Base(os.Args[0]), err)
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()
	if err := tarindex.WriteIndex(os.Args[2], f); err != nil {
		_ = f.Close()
		_ = os.Remove(os.Args[1])
		_, _ = fmt.Fprintf(os.Stderr, "%s: Error on source directory: %s\n", path.Base(os.Args[0]), err)
		os.Exit(1)
	}
	os.Exit(0)
}
