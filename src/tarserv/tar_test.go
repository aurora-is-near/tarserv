package tarserv

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestDir(t *testing.T) {
	f, _ := ioutil.TempFile(os.TempDir(), "test_*.tar")
	defer func() { _ = f.Close() }()
	log.Println(f.Name())
	buf := new(bytes.Buffer)
	buf.WriteString("Nothing")
	if err := NewTar(os.TempDir(), f, OptRebase("/root/")); err != nil {
		// if err := NewTar(os.TempDir(), f, OptRebase("/root/"), OptByteSeek("/root/caffeine1001.pid", 0)); err != nil {
		t.Fatalf("NewTar: %s", err)
	}
}
