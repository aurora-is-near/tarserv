package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aurora-is/tarserv/src/tarserv"
)

var (
	sourceDir string
	prefix    string
	address   string
)

func init() {
	flag.StringVar(&sourceDir, "source", "/var/data/snapshots/", "source directory")
	flag.StringVar(&prefix, "prefix", "/snapshots/", "url path prefix")
	flag.StringVar(&address, "listen", "127.0.0.1:9876", "ip:port to listen")
}

func main() {
	flag.Parse()
	if err := tarserv.Serve(address, prefix, sourceDir, ".tarserv_version"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}
