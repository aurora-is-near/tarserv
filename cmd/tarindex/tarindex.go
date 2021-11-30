package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/aurora-is-near/tarserv/src/tarindex"

	"github.com/aurora-is-near/tarserv/src/util"
)

var (
	referenceFile      string
	bytepos            int64
	byteend            int64
	postfixFileName    string
	postfixFileContent string

	postfixFile *tarindex.PostfixFile
)

func init() {
	flag.StringVar(&referenceFile, "f", "", "optional reference file in tar archive")
	flag.Int64Var(&bytepos, "p", 0, "optional byte seek position")
	flag.Int64Var(&byteend, "e", 0, "optional byte seek end position")
	flag.StringVar(&postfixFileName, "n", "", "Name for postfix file")
	flag.StringVar(&postfixFileContent, "c", "", "Content of postfix file")
}

func main() {
	flag.Parse()
	args := flag.Args()
	outWriter := os.Stdout
	if len(args) < 1 {
		_, _ = fmt.Fprintf(os.Stderr, "%s <indexfile> [<destination tarfile>]\n", path.Base(os.Args[0]))
		os.Exit(1)
	}
	index, err := os.Open(args[0])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: Error opening index file: %s\n", path.Base(os.Args[0]), err)
		os.Exit(1)
	}
	defer func() { _ = index.Close() }()
	if len(args) > 1 {
		if args[1] != "-" {
			out, err := util.CreateFile(args[1])
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%s: Error opening output file: %s\n", path.Base(os.Args[0]), err)
				os.Exit(1)
			}
			defer func() { _ = out.Close() }()
			outWriter = out
		}
	}
	if postfixFileName != "" {
		postfixFile = &tarindex.PostfixFile{
			Name:    postfixFileName,
			Content: []byte(postfixFileContent),
		}
	}
	idx, err := tarindex.NewIndexReader(index, outWriter, postfixFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: Error opening index file: %s\n", path.Base(os.Args[0]), err)
		os.Exit(1)
	}
	_, err = idx.SeekAndWrite(referenceFile, bytepos, byteend)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: Error encoding tar stream: %s\n", path.Base(os.Args[0]), err)
		os.Exit(1)
	}
	os.Exit(0)
}
