package tarserv

import (
	"archive/tar"
	"errors"
	"io"
	"log"
	"os"
	"path"
)

// ErrNoDir is returned if a given path is not a directory.
var ErrNoDir = errors.New("no directory")

// PathFunc creates a function for rewriting paths.
type PathFunc func(base string) PathRewriteFunc

// PathRewriteFunc rewrites a path.
type PathRewriteFunc func(d string) string

type activeConfig struct {
	PathType       PathFunc
	ActivePathType PathRewriteFunc
	HeaderFixes    []headerFixFunc
	Appends        []appendFileOpt
	mockwrite      bool
	skipOutBytes   int64
	startFile      string
}

type tarCreator struct {
	options   activeConfig
	out       *tar.Writer
	dontWrite bool
	fromFile  string
}

// ToDo: SkipWriter: Skip first x bytes of writes before writing to underlying writer.
// ToDo: IndexWriter: Count bytes, write filename & current bytes on flush event.
// ToDo: Use filelist from index-writer as input.
// Index:   StartByte path  (use path to discover if this is dir/link/file)
// Tar file-header is 512 bytes, file content is rounded to 512b blocks, end of tar is two empty 512b blocks

// NewTar creates a tar stream written to w that contains dir. It will have paths as determined by pathType.
func NewTar(dir string, w io.Writer, options ...Option) error {
	creator := new(tarCreator)
	appliedOptions := newOptions()
	for _, opt := range options {
		opt.applyOption(appliedOptions)
	}

	appliedOptions.ActivePathType = appliedOptions.PathType(dir)
	if appliedOptions.mockwrite {
		creator.dontWrite = true
		creator.fromFile = appliedOptions.startFile
	}
	creator.options = *appliedOptions
	creator.out = tar.NewWriter(w)
	if err := creator.addDir(dir); err != nil {
		return err
	}
	creator.addAppends(dir)
	return creator.out.Close()
}

func (creator *tarCreator) mockWrite(hdr *tar.Header) bool {
	if creator.dontWrite {
		if hdr.Name == creator.fromFile {
			creator.dontWrite = false
		}
	}
	return creator.dontWrite
}

func (creator *tarCreator) addAppends(dir string) {
	for _, ap := range creator.options.Appends {
		if err := ap.Append(dir, creator.out, creator.options); err != nil {
			log.Printf("Append failed '%s': %s", ap.name, err)
		}
	}
}

func fixHeader(header *tar.Header, options activeConfig) {
	for _, fix := range options.HeaderFixes {
		fix(header)
	}
}

func isRegular(fi os.FileInfo) bool {
	mode := fi.Mode()
	return mode & ^os.ModeType == mode
}

func isLink(fi os.FileInfo) bool {
	mode := fi.Mode()
	return mode&os.ModeSymlink != 0
}

func (creator *tarCreator) addLink(name string, fi os.FileInfo) error {
	link, err := os.Readlink(name)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return err
	}
	header.Name = creator.options.ActivePathType(name)
	fixHeader(header, creator.options)
	if creator.mockWrite(header) {
		return nil
	}
	if err := creator.out.WriteHeader(header); err != nil {
		return err
	}
	return nil
}

func (creator *tarCreator) addFile(name string, fi os.FileInfo) error {
	header, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	header.Name = creator.options.ActivePathType(name)
	fixHeader(header, creator.options)
	if creator.mockWrite(header) {
		return nil
	}
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if err := creator.out.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(creator.out, io.LimitReader(f, fi.Size()))
	return err
}

func (creator *tarCreator) addDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	stat, err := d.Stat()
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return ErrNoDir
	}
	header, err := tar.FileInfoHeader(stat, "")
	if err != nil {
		return err
	}
	header.Name = creator.options.ActivePathType(dir)
	fixHeader(header, creator.options)
	if !creator.mockWrite(header) {
		if err := creator.out.WriteHeader(header); err != nil {
			return err
		}
	}
DirLoop:
	for {
		entries, _ := d.Readdir(10)
		if entries == nil || len(entries) == 0 {
			break DirLoop
		}
	EntryLoop:
		for _, e := range entries {
			name := path.Join(dir, e.Name())
			if e.IsDir() {
				if err := creator.addDir(name); err != nil {
					log.Printf("Failed to add dir '%s': %s", name, err)
					continue EntryLoop
				}
				continue EntryLoop
			}
			if isLink(e) {
				if err := creator.addLink(name, e); err != nil {
					log.Printf("Failed to add link '%s': %s", name, err)
					continue EntryLoop
				}
				continue EntryLoop
			}
			if isRegular(e) {
				if err := creator.addFile(name, e); err != nil {
					log.Printf("Failed to add file '%s': %s", name, err)
					continue EntryLoop
				}
				continue EntryLoop
			}
		}
	}
	return nil
}
