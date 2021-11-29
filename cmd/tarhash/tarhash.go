package main

import (
	"fmt"
	"os"
	"path"

	"github.com/aurora-is/tarserv/src/splitting"
)

func main() {
	out := os.Stdout
	if len(os.Args) != 3 {
		_, _ = fmt.Fprintf(os.Stderr, "%s <input.tar> <output.hashfile>\n", path.Base(os.Args[0]))
		os.Exit(1)
	}
	if os.Args[2] != "-" {
		of, err := os.Create(os.Args[2])
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s ERROR: %s\n", path.Base(os.Args[0]), err)
			os.Exit(1)
		}
		defer of.Close()
		out = of
	}
	if err := splitting.ReadSHA256(os.Args[1], out); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s ERROR: %s\n", path.Base(os.Args[0]), err)
		os.Exit(1)
	}
	os.Exit(0)
}
