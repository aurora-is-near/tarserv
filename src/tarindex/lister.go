// Package tarindex lists the content of a directory and its subdirectories, including only directories, links and files.
// It produces a stream of paths and file sizes.
package tarindex

import (
	"os"
	"path"
	"sync/atomic"
)

type lister struct {
	c     chan interface{}
	close int32
}

func (list *lister) closed() bool {
	return atomic.LoadInt32(&list.close) != 0
}

func (list *lister) exit() {
	atomic.StoreInt32(&list.close, 1)
}

func newLister() *lister {
	return &lister{
		c: make(chan interface{}, 10),
	}
}

func (list *lister) closeChan() {
	if list.c != nil {
		close(list.c)
		list.c = nil
	}
}

func (list *lister) sendEntry(name string, entryType EntryType, size int64) {
	if list.closed() {
		list.closeChan()
		return
	}
	list.c <- &ListEntry{
		Size: size,
		Name: name,
		Type: entryType,
	}
}

func (list *lister) addDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	list.sendEntry(dir, EntryTypeDirectory, 0)
DirLoop:
	for {
		if list.closed() {
			return nil
		}
		entries, _ := d.Readdir(10)
		if entries == nil || len(entries) == 0 {
			break DirLoop
		}
	EntryLoop:
		for _, e := range entries {
			if list.closed() {
				return nil
			}
			name := path.Join(dir, e.Name())
			switch {
			case e.IsDir():
				if err := list.addDir(name); err != nil {
					// log.Printf("Failed to list dir '%s': %s", name, err)
					continue EntryLoop
				}
			case isLink(e):
				list.sendEntry(name, EntryTypeLink, 0)
			case isRegular(e):
				list.sendEntry(name, EntryTypeFile, e.Size())
			}
		}
	}
	return nil
}

func mkListEntry(dir string, fi os.FileInfo) *ListEntry {
	ret := new(ListEntry)
	switch {
	case fi.IsDir():
		ret.Name = dir
		ret.Type = EntryTypeDirectory
	case isLink(fi):
		ret.Name = path.Join(dir, fi.Name())
		ret.Type = EntryTypeLink
	case isRegular(fi):
		ret.Name = path.Join(dir, fi.Name())
		ret.Type = EntryTypeFile
		ret.Size = fi.Size()
	default:
		return nil
	}
	return ret
}
