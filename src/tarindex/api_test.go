package tarindex

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestWriter(t *testing.T) {
	//goland:noinspection SpellCheckingInspection
	f, err := ioutil.TempFile(os.TempDir(), "tarindex.")
	if err != nil {
		t.Fatalf("TempFile: %s", err)
	}

	if err := WriteIndex(os.TempDir(), f); err != nil {
		t.Fatalf("WriteIndex: %s", err)
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	_ = f.Close()

	f, err = os.Open(name)
	if err != nil {
		t.Fatalf("ReadFile: %s", err)
	}
	defer func() { _ = f.Close() }()
	size, name, err := IndexHeader(f)
	if err != nil {
		t.Fatalf("IndexHeader: %s", err)
	}
	if name != os.TempDir() {
		t.Error("Wrong name")
	}
	if size == 0 {
		t.Error("Header size not written")
	}
}

func TestLister(t *testing.T) {
	var offset int64
	var offsetRead int64
	entryFunc := func(e *ListEntry) error {
		var entry *BinaryEntry
		offset, entry = e.BinaryEntry(offset)
		re := entry.ToListEntry(offsetRead)
		offsetRead = re.LastByte
		return nil
	}
	if err := ListToFunc(os.TempDir(), entryFunc); err != nil {
		t.Fatalf("ListToFunc: %s", err)
	}
}
