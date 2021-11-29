package tarindex

import (
	"errors"
	"io"
	"path"
)

var (
	ErrMissingHeader = errors.New("missing header")
)

func listToChan(dir string) (list *lister) {
	dir = path.Clean(dir)
	list = newLister()
	go func() {
		defer list.closeChan()
		if err := list.addDir(dir); err != nil {
			list.c <- err
		}
	}()
	return list
}

// ListToChan produces a flow of list entries send to chan entries.
// The channel is closed after listing has been completed.
// The channel will contain either *ListEntry or error entries.
//goland:noinspection GoUnusedExportedFunction
func ListToChan(dir string) (entries chan interface{}) {
	list := listToChan(dir)
	return list.c
}

// ListToFunc produces a flow of list entries that are given to entryFunc for processing.
func ListToFunc(dir string, entryFunc func(*ListEntry) error) error {
	list := listToChan(dir)
	for m := range list.c {
		switch n := m.(type) {
		case *ListEntry:
			if err := entryFunc(n); err != nil {
				list.exit()
				return err
			}
		case error:
			return n
		}
	}
	return nil
}

// IndexHeader reads the header of the index file and returns the total size (without postfix files),and the root directory.
// In case of ErrMissingHeader only the root directory is usable. The filesize is indeterminate.
func IndexHeader(r io.Reader) (size int64, dir string, err error) {
	if w2, ok := r.(io.ReadSeeker); ok {
		if _, err := w2.Seek(0, io.SeekStart); err != nil {
			return 0, "", err
		}
	}
	buf := new(BinaryEntry)
	if _, err := r.Read(buf[:]); err != nil {
		return 0, "", err
	}
	e := buf.ToListEntry(0)
	if e.Type != EntryTypeHeader {
		return 0, e.Name, ErrMissingHeader
	}
	return e.Size, e.Name, nil
}

// WriteIndex writes an index file. w should be an io.WriteSeeker if possible.
func WriteIndex(dir string, w io.Writer) error {
	var offset int64
	var fileHdr, hdr *BinaryEntry

	_, fileHdr = (&ListEntry{
		Type: EntryTypeHeader,
		Name: dir,
	}).BinaryEntry(0)

	entryFunc := func(e *ListEntry) error {
		if fileHdr != nil {
			if _, err := w.Write(fileHdr[:]); err != nil {
				return err
			}
			fileHdr = nil
		}
		offset, hdr = e.BinaryEntry(offset)
		if _, err := w.Write(hdr[:]); err != nil {
			return err
		}
		return nil
	}
	err := ListToFunc(dir, entryFunc)
	if err == nil {
		if w2, ok := w.(io.WriteSeeker); ok {
			if _, err := w2.Seek(0, io.SeekStart); err != nil {
				return err
			}
			_, fileHdr = (&ListEntry{
				Type: EntryTypeHeader,
				Name: dir,
			}).BinaryEntry(offset + tarFooterSize)
			if _, err := w.Write(fileHdr[:]); err != nil {
				return err
			}
		}
	}
	return err
}
