package tarindex

import (
	"testing"
)

func TestFixPath(t *testing.T) {
	fp := &PathMod{
		BaseDir: "/tmp/",
		ModDir:  "./",
	}
	n := fp.FixPath("/tmp/something")
	if n != "./something" {
		t.Errorf("Failed: %s != %s", n, "./something")
	}

}
