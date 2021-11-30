package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/aurora-is-near/tarserv/src/deliver"
)

var (
	indexDir      string
	listenAddress string
	prefix        string
)

func init() {
	flag.StringVar(&indexDir, "i", "/var/snapshots/", "Directory containing index files produced by tarindex.")
	flag.StringVar(&listenAddress, "l", "127.0.0.1:18123", "IP:Port to listen on.")
	flag.StringVar(&prefix, "p", "/", "Request path.")
}

func main() {
	flag.Parse()
	h := &deliver.TarHandler{
		IndexDirectory: indexDir,
	}
	mux := http.NewServeMux()
	mux.Handle(prefix, http.StripPrefix(prefix, h))
	log.Println("Starting...")
	go func() {
		if err := http.ListenAndServe(listenAddress, mux); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to listen: %s", err)
			os.Exit(1)
		}
	}()
	log.Println("Running")
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	<-c
	log.Println("Stop")
}
