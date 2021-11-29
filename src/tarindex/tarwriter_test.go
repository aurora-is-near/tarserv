package tarindex

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

func TestClose(t *testing.T) {
	for skipbytes := int64(0); skipbytes < tarBlockSize*2; skipbytes++ {
		buf := new(bytes.Buffer)
		w := NewTarWriter(buf)
		n, err := w.Close(skipbytes, 10000)
		if err != nil {
			t.Errorf("Write %d: %s", skipbytes, err)
		}
		if n+skipbytes != tarBlockSize*2 {
			t.Errorf("Not two zero blocks calculated: %d %d", skipbytes, n+skipbytes)
		}
		if int64(buf.Len()) != (tarBlockSize*2)-skipbytes {
			t.Errorf("Not two zero blocks written: %d %d", skipbytes, buf.Len())
		}
	}
}

func TestLink(t *testing.T) {
	fixModTime = func(x time.Time) time.Time { return time.Time{} }
	tdirName, err := ioutil.TempDir(os.TempDir(), "tarwriter.")
	if err != nil {
		t.Fatalf("TempDir: %s", err)
	}
	defer func() { _ = os.RemoveAll(tdirName) }()
	linkDest := path.Join(tdirName, "linkdest")
	linkSource := path.Join(tdirName, "linksource")
	f, err := os.Create(linkDest)
	if err != nil {
		t.Fatalf("Create: %s", err)
	}
	_ = f.Close()
	if err := os.Symlink(linkDest, linkSource); err != nil {
		t.Fatalf("Link: %s", err)
	}
	buf := new(bytes.Buffer)
	w := tar.NewWriter(buf)
	fi, _ := os.Lstat(linkSource)
	hdr, _ := tar.FileInfoHeader(fi, linkDest)
	(&TarWriter{}).fixHeader(hdr)
	hdr.Name = path.Join(tdirName, hdr.Name)
	_ = w.WriteHeader(hdr)
	_ = w.Close()
	td := buf.Bytes()
	if len(td) != int(tarBlockSize*3) {
		t.Error("Error creating testdata")
	}
	for skipbytes := int64(0); skipbytes <= tarBlockSize; skipbytes++ {
		buf := new(bytes.Buffer)
		tarW := NewTarWriter(buf)
		entry := mkListEntry(tdirName, fi)
		if _, err := tarW.writeLinkEntry(entry, skipbytes, 100000); err != nil {
			t.Fatalf("writeLinkEntry: %s", err)
		}
		_, _ = tarW.Close(0, 100000)
		td2 := buf.Bytes()
		if len(td[skipbytes:]) != len(td2) {
			t.Errorf("Sizes dont match: %d != %d", len(td[skipbytes:]), len(td2))
		}
		if !bytes.Equal(td[skipbytes:], td2) {
			t.Errorf("Not equal: %d", skipbytes)
		}
	}
}

func TestDir(t *testing.T) {
	fixModTime = func(x time.Time) time.Time { return time.Time{} }
	tdirName, err := ioutil.TempDir(os.TempDir(), "tarwriter.")
	if err != nil {
		t.Fatalf("TempDir: %s", err)
	}
	defer func() { _ = os.RemoveAll(tdirName) }()
	buf := new(bytes.Buffer)
	w := tar.NewWriter(buf)
	fi, _ := os.Stat(tdirName)
	hdr, _ := tar.FileInfoHeader(fi, "")
	(&TarWriter{}).fixHeader(hdr)
	hdr.Name = tdirName
	_ = w.WriteHeader(hdr)
	_ = w.Close()
	td := buf.Bytes()
	if len(td) != int(tarBlockSize*3) {
		t.Error("Error creating testdata")
	}
	for skipbytes := int64(0); skipbytes <= tarBlockSize; skipbytes++ {
		buf := new(bytes.Buffer)
		tarW := NewTarWriter(buf)
		entry := mkListEntry(tdirName, fi)
		if _, err := tarW.writeDirectoryEntry(entry, skipbytes, 100000); err != nil {
			t.Fatalf("writeDirectoryEntry: %s", err)
		}
		_, _ = tarW.Close(0, 100000)
		td2 := buf.Bytes()
		if len(td[skipbytes:]) != len(td2) {
			t.Errorf("Sizes dont match: %d != %d", len(td[skipbytes:]), len(td2))
		}
		if !bytes.Equal(td[skipbytes:], td2) {
			t.Errorf("Not equal: %d", skipbytes)
		}
	}
}

func TestFile(t *testing.T) {
	fixModTime = func(x time.Time) time.Time { return time.Time{} }
	f, err := ioutil.TempFile(os.TempDir(), "tarWriter.")
	if err != nil {
		t.Fatalf("TempFile: %s", err)
	}
	name := f.Name()
	if _, err := io.Copy(f, io.LimitReader(rand.Reader, 10+tarBlockSize*3)); err != nil {
		t.Fatalf("Copy: %s", err)
	}
	_ = f.Close()
	defer func() { _ = os.Remove(name) }()
	buf := new(bytes.Buffer)
	w := tar.NewWriter(buf)
	fi, _ := os.Stat(name)
	hdr, _ := tar.FileInfoHeader(fi, "")
	hdr.Name = path.Join(os.TempDir(), hdr.Name)
	(&TarWriter{}).fixHeader(hdr)
	_ = w.WriteHeader(hdr)
	d, _ := ioutil.ReadFile(name)
	_, _ = w.Write(d)
	_ = w.Close()
	td := buf.Bytes()
	for skipbytes := int64(0); skipbytes < tarBlockSize*5; skipbytes++ {
		buf := new(bytes.Buffer)
		tarW := NewTarWriter(buf)
		entry := mkListEntry(os.TempDir(), fi)
		if _, err := tarW.writeFileEntry(entry, skipbytes, 100000); err != nil {
			t.Fatalf("writeFileEntry: %s", err)
		}
		_, _ = tarW.Close(0, 100000)
		td2 := buf.Bytes()
		if len(td[skipbytes:]) != len(td2) {
			t.Errorf("Sizes dont match (%d): %d != %d", skipbytes, len(td[skipbytes:]), len(td2))
		}
		if !bytes.Equal(td[skipbytes:], td2) {
			t.Errorf("Not equal: %d", skipbytes)
		}
	}
}
