package tarindex

import (
	"archive/tar"
	"errors"
	"io"
	"os"
	"time"
)

// Required for testing: Replace with func(x time.Time) time.Time { return time.Time{} }
var fixModTime = func(t time.Time) time.Time { return t }

var (
	ErrIndexFSMismatch = errors.New("index does not match filesystem")
	ErrUnsupported     = errors.New("unsupported filetype")
	ErrSkipBoundary    = errors.New("skip beyond file boundary")
)

func minNotNegativeA(a, b int64) int64 {
	if a < b && a >= 0 {
		return a
	}
	return b
}

func maxBytes(d []byte, maxbytes int64) []byte {
	l := minNotNegativeA(maxbytes, int64(len(d)))
	if l == int64(len(d)) {
		return d
	}
	return d[:l]
}

type TarWriter struct {
	w       io.Writer
	FixPath func(string) string
}

func NewTarWriter(w io.Writer) *TarWriter {
	return &TarWriter{w: w}
}

func (tw *TarWriter) fixPath(path string) string {
	if tw.FixPath == nil {
		return path
	}
	return tw.FixPath(path)
}

func (tw *TarWriter) fixHeader(hdr *tar.Header) {
	hdr.Name = tw.fixPath(hdr.Name)
	hdr.Gname = ""
	hdr.Uname = ""
	hdr.Uid = 0
	hdr.Gid = 0
	hdr.ModTime = fixModTime(hdr.ModTime)
}

func (tw *TarWriter) fixLink(link string) string {
	return tw.fixPath(link)
}

// WriteEntry writes e's tar entry (header and content) to w. It skips the first skipbytes bytes.
func (tw *TarWriter) WriteEntry(e *ListEntry, skipbytes, maxbytes int64) (int64, error) {
	switch e.Type {
	case EntryTypeDirectory:
		return tw.writeDirectoryEntry(e, skipbytes, maxbytes)
	case EntryTypeLink:
		return tw.writeLinkEntry(e, skipbytes, maxbytes)
	case EntryTypeFile:
		return tw.writeFileEntry(e, skipbytes, maxbytes)
	default:
		return 0, ErrUnsupported
	}
}

func (tw *TarWriter) writeDirectoryEntry(e *ListEntry, skipbytes, maxbytes int64) (int64, error) {
	if skipbytes < 0 {
		skipbytes = 0
	}
	if skipbytes > tarHeaderSize {
		panic("Directory with skipbytes>tarHeaderBytesFromFileInfo")
	}
	fi, err := os.Stat(e.Name)
	if err != nil {
		return 0, err
	}
	if !fi.IsDir() {
		return 0, ErrIndexFSMismatch
	}
	hdr, err := tarHeaderBytesFromFileInfo(e, fi, "", tw.fixHeader)
	if err != nil {
		return 0, err
	}
	n, err := tw.w.Write(maxBytes(hdr[skipbytes:], maxbytes))
	return int64(n), err
}

func (tw *TarWriter) writeLinkEntry(e *ListEntry, skipbytes, maxbytes int64) (int64, error) {
	if skipbytes < 0 {
		skipbytes = 0
	}
	if skipbytes > tarHeaderSize {
		panic("Link with skipbytes>tarHeaderBytesFromFileInfo")
	}
	fi, err := os.Lstat(e.Name)
	if err != nil {
		return 0, err
	}
	if !isLink(fi) {
		return 0, ErrIndexFSMismatch
	}
	link, err := os.Readlink(e.Name)
	if err != nil {
		return 0, err
	}
	hdr, err := tarHeaderBytesFromFileInfo(e, fi, tw.fixLink(link), tw.fixHeader)
	if err != nil {
		return 0, err
	}
	n, err := tw.w.Write(maxBytes(hdr[skipbytes:], maxbytes))
	return int64(n), err
}

func paddingSize(size int64) int64 {
	r := size % tarBlockSize
	if r == 0 {
		return 0
	}
	r = tarBlockSize - r
	return r
}

func (tw *TarWriter) writeFileEntry(e *ListEntry, skipbytes, maxbytes int64) (int64, error) {
	var nHeader, nBody, nPad int64
	if skipbytes < 0 {
		skipbytes = 0
	}
	fi, err := os.Stat(e.Name)
	if err != nil {
		return 0, err
	}
	if !isRegular(fi) {
		return 0, ErrIndexFSMismatch
	}
	fileSize := fi.Size()
	pad := paddingSize(fileSize)
	if tarHeaderSize+fileSize+pad < skipbytes {
		return 0, ErrSkipBoundary
	}
	f, err := os.Open(e.Name)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	if skipbytes <= tarHeaderSize {
		var n int
		hdr, err := tarHeaderBytesFromFileInfo(e, fi, "", tw.fixHeader)
		if err != nil {
			return 0, err
		}
		if n, err = tw.w.Write(maxBytes(hdr[skipbytes:], maxbytes)); err != nil {
			return int64(n), err
		}
		nHeader = int64(n)
		skipbytes = 0
		maxbytes -= nHeader
		if maxbytes == 0 {
			return nHeader, nil
		}
	} else if skipbytes > 0 {
		skipbytes -= tarHeaderSize
	}
	if skipbytes <= fileSize {
		if _, err := f.Seek(skipbytes, io.SeekStart); err != nil {
			return nHeader, err
		}
		nBody, err = io.Copy(tw.w, io.LimitReader(f, maxbytes))
		if err != nil {
			return nBody + nHeader, err
		}
		maxbytes -= nBody
		if maxbytes == 0 {
			return nBody + nHeader, nil
		}
		skipbytes = 0
	} else {
		skipbytes -= fileSize
	}
	if pad > 0 {
		var n int
		n, err = tw.w.Write(maxBytes((zeroBlock[:pad])[skipbytes:], maxbytes))
		nPad = int64(n)
		maxbytes -= nPad
		if maxbytes == 0 {
			return nBody + nHeader + nPad, err
		}
	}
	return nBody + nHeader + nPad, err
}

// PostfixFileSize is the size of a file with the given content.
func PostfixFileSize(content []byte) int64 {
	fileSize := int64(len(content))
	pad := paddingSize(fileSize)
	return tarHeaderSize + fileSize + pad
}

// AddPostfixFile adds a file with name and content to the archive.
func (tw *TarWriter) AddPostfixFile(name string, content []byte, skipbytes, maxbytes int64) (int64, error) {
	var nHeader, nContent, nPad int64
	if skipbytes < 0 {
		skipbytes = 0
	}
	fileSize := int64(len(content))
	if skipbytes <= tarHeaderSize {
		var n int
		now := time.Now()
		header := &tar.Header{
			Name:       name,
			Typeflag:   tar.TypeReg,
			Size:       fileSize,
			Mode:       int64(0600),
			ModTime:    now,
			AccessTime: now,
			ChangeTime: now,
		}
		hdr, err := tarHeaderBytes(header, tw.fixHeader)
		if err != nil {
			return 0, err
		}
		if n, err = tw.w.Write(maxBytes(hdr[skipbytes:], maxbytes)); err != nil {
			return int64(n), err
		}
		nHeader = int64(n)
		skipbytes = 0
	} else if skipbytes > 0 {
		skipbytes -= tarHeaderSize
	}
	if skipbytes < fileSize {
		content = content[skipbytes:]
		skipbytes = 0
	} else {
		skipbytes -= fileSize
	}
	n, err := tw.w.Write(content)
	nContent = int64(n)
	if err != nil {
		return nHeader + nContent, err
	}
	if pad := paddingSize(fileSize); pad > 0 {
		var n int
		n, err = tw.w.Write(maxBytes((zeroBlock[:pad])[skipbytes:], maxbytes))
		nPad = int64(n)
	}
	return nHeader + nContent + nPad, err
}

// Close the archive.
func (tw *TarWriter) Close(skipbytes, maxbytes int64) (int64, error) {
	var written int64
	var n int
	var err error
	if skipbytes < 0 {
		skipbytes = 0
	}

	if skipbytes > tarBlockSize*2 {
		panic("Footer with skipbytes>tarBlockSize*2")
	}
	writeBytes := minNotNegativeA(maxbytes, (tarBlockSize*2)-skipbytes)
	for i := int64(1); i < 3; i++ {
		if writeBytes == 0 {
			break
		}
		if writeBytes >= tarBlockSize {
			n, err = tw.w.Write(zeroBlock[:])
			written += int64(n)
			if err != nil {
				return written, err
			}
			writeBytes -= tarBlockSize
			continue
		} else {
			n, err = tw.w.Write(zeroBlock[:writeBytes])
			written += int64(n)
			break
		}
	}
	return written, err
}
