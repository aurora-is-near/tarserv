package main

import (
	"fmt"
	"os"
	"path"

	"github.com/aurora-is/tarserv/src/splitting"
)

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "%s <input.tar>\n", path.Base(os.Args[0]))
		os.Exit(1)
	}
	if err := splitting.SplitTarMiddle(os.Args[1]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s ERR: %s\n", path.Base(os.Args[0]), err)
		os.Exit(1)
	}
	os.Exit(0)
}
