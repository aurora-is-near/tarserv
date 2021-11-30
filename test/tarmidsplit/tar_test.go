package tarmidsplit

import (
	"os"
	"testing"

	"github.com/aurora-is-near/tarserv/src/splitting"
)

func TestTar(t *testing.T) {
	fn := "/tmp/linux-5.15.4.tar"
	if err := splitting.SplitTarMiddle(fn); err != nil {
		t.Fatalf("SplitTarMiddle: %s", err)
	}
}

func TestHash(t *testing.T) {
	fn := "/tmp/linux-5.15.4.tar"
	out, _ := os.Create(fn + ".hashes")
	defer out.Close()
	if err := splitting.ReadSHA256(fn, out); err != nil {
		t.Fatalf("ReadSHA256: %s", err)
	}
}
