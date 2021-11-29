package tarindex

import (
	"errors"
	"io"
	"path"
)

var (
	// ErrNoSeek is returned if multiple seeks in the index are attempted.
	ErrNoSeek = errors.New("no more seeks")
	// ErrMissingFile is returned when a reference file does not exist in the index.
	ErrMissingFile = errors.New("reference file not found")
)

// IndexReader parses a tar index and produces a (partial) tar stream.
type IndexReader struct {
	r           io.Reader
	totalSize   int64
	baseDir     string
	postFixFile *PostfixFile
	w           *TarWriter

	seekEntry  *ListEntry // From searching. Entry that contains next byte.
	seekOffset int64      // offset encountered while seeking.
	skipBytes  int64      // skipBytes number of bytes to skip on seekEntry.

	noMoreSeek bool // Set to true if more seeks are impossible.
}

// PostfixFile is a file that may be generated at the end of the tar stream.
type PostfixFile struct {
	Name    string
	Content []byte
}

// NewIndexReader creates an IndexReader that reads the index from r and writes the tar stream to w. It may attach
// a postFixFile.
func NewIndexReader(r io.Reader, w io.Writer, postFixFile *PostfixFile) (*IndexReader, error) {
	size, dir, err := IndexHeader(r)
	if err != nil {
		if err != ErrMissingHeader {
			return nil, err
		}
		size = 0
	} else {
		if postFixFile != nil {
			size += PostfixFileSize(postFixFile.Content)
		}
	}
	ir := &IndexReader{
		r:           r,
		totalSize:   size,
		baseDir:     dir,
		postFixFile: postFixFile,
		w:           NewTarWriter(w),
	}
	ir.w.FixPath = PathMod{BaseDir: ir.baseDir, ModDir: "./"}.FixPath
	return ir, nil
}

// Size returns the total size of the tar stream, if known, otherwise 0.
func (ir *IndexReader) Size() int64 {
	return ir.totalSize
}

// SeekAndWrite generates a tar stream starting at either:
// - pos bytes from beginning of index if filename is empty.
// - pos bytes from the beginning of the file with name filename.
// If pos == 0 the complete tar stream is written.
// maxbytes limits how many bytes starting from the beginning of the archive should be written.
func (ir *IndexReader) SeekAndWrite(filename string, pos, maxbytes int64, informFunc ...func(maxBytes int64)) (int64, error) {
	if pos > 0 && filename == "" {
		if err := ir.SeekByte(pos); err != nil {
			return 0, err
		}
	} else if filename != "" {
		if err := ir.SeekFile(filename, pos); err != nil {
			return 0, err
		}
	}
	if maxbytes > 0 {
		maxbytes = maxbytes - pos
	} else if ir.totalSize > 0 {
		maxbytes = ir.totalSize - pos
	} else {
		maxbytes = -1
	}
	if len(informFunc) == 1 {
		contentLength := func() int64 {
			offset := ir.seekOffset + ir.skipBytes
			if offset > 0 {
				return tarBlockSize + ir.totalSize - offset
			}
			return ir.totalSize
		}()
		informFunc[0](contentLength)
	}
	return ir.WriteTar(maxbytes)
}

// SeekByte seeks through index to find the matching entry from which to produce the tar stream.
func (ir *IndexReader) SeekByte(pos int64) error {
	var offset int64
	if pos == 0 {
		return nil
	}
	if ir.totalSize != 0 && ir.totalSize < pos {
		return ErrSkipBoundary
	}
	if ir.noMoreSeek {
		return ErrNoSeek
	}
	ir.noMoreSeek = true
IndexLoop:
	for {
		buf := new(BinaryEntry)
		if _, err := ir.r.Read(buf[:]); err != nil {
			if err == io.EOF {
				break IndexLoop
			}
			return err
		}
		entry := buf.ToListEntry(offset)
		offset = entry.LastByte
		if entry.LastByte > pos {
			ir.skipBytes = pos - entry.FirstByte
			ir.seekEntry = entry
			ir.seekOffset = offset
			return nil
		}
	}
	ir.seekOffset = offset
	ir.skipBytes = pos - offset
	size := offset
	if ir.postFixFile != nil {
		size += PostfixFileSize(ir.postFixFile.Content)
	}
	size += tarFooterSize
	if pos > size {
		return ErrSkipBoundary
	}
	return nil
}

func (ir *IndexReader) matchPath(name, match string) bool {
	return path.Clean(ir.w.FixPath(name)) == path.Clean(match)
}

// SeekFile seeks through index to find the matching entry for filename, and then seeks pos bytes from there.
func (ir *IndexReader) SeekFile(filename string, pos int64) error {
	var offset int64
	var fileFound bool
	if filename == "" {
		if pos > 0 {
			return ir.SeekByte(pos)
		}
		return nil
	}

	if ir.totalSize != 0 && ir.totalSize < pos {
		return ErrSkipBoundary
	}
	if ir.noMoreSeek {
		return ErrNoSeek
	}
	ir.noMoreSeek = true
IndexLoop:
	for {
		buf := new(BinaryEntry)
		if _, err := ir.r.Read(buf[:]); err != nil {
			if err == io.EOF {
				break IndexLoop
			}
			return err
		}
		entry := buf.ToListEntry(offset)
		offset = entry.LastByte
		if !fileFound && ir.matchPath(entry.Name, filename) {
			// File found. Only first match is considered.
			fileFound = true
			pos = entry.FirstByte + pos // Calculate pos relative to file entry.
		}
		if fileFound && entry.LastByte > pos {
			// Byte found.
			ir.skipBytes = pos - entry.FirstByte
			ir.seekEntry = entry
			ir.seekOffset = offset
			return nil
		}
	}
	if !fileFound {
		return ErrMissingFile
	}
	// Not found, match must be in postfix file or end-of-file padding.
	ir.seekOffset = offset
	ir.skipBytes = pos - offset
	size := offset
	if ir.postFixFile != nil {
		size += PostfixFileSize(ir.postFixFile.Content)
	}
	size += tarFooterSize
	if pos > size {
		return ErrSkipBoundary
	}
	return nil
}

// WriteTar writes a (partial) tar stream from the current seek position.
func (ir *IndexReader) WriteTar(maxbytes int64) (int64, error) {
	var written int64
	var n int64
	var err error
	if maxbytes == 0 {
		return written, nil
	}
	offset := ir.seekOffset
	if ir.seekEntry != nil {
		if n, err = ir.w.WriteEntry(ir.seekEntry, ir.skipBytes, maxbytes); err != nil {
			return n, err
		}
		ir.skipBytes = 0
		written += n
		maxbytes -= n
		if maxbytes == 0 {
			return written, nil
		}
	}
	if ir.skipBytes == 0 {
	IndexLoop:
		for {
			buf := new(BinaryEntry)
			if _, err := ir.r.Read(buf[:]); err != nil {
				if err == io.EOF {
					break IndexLoop
				}
				return written, err
			}
			entry := buf.ToListEntry(offset)
			offset = entry.LastByte
			if n, err = ir.w.WriteEntry(entry, 0, maxbytes); err != nil {
				return n + written, err
			}
			written += n
			maxbytes -= n
			if maxbytes == 0 {
				return written, nil
			}
		}
	}
	if ir.postFixFile != nil {
		postfixSize := PostfixFileSize(ir.postFixFile.Content)
		if ir.skipBytes < postfixSize {
			if n, err = ir.w.AddPostfixFile(ir.postFixFile.Name, ir.postFixFile.Content, ir.skipBytes, maxbytes); err != nil {
				return n + written, err
			}
			written += n
			maxbytes -= n
			ir.skipBytes = 0
			if maxbytes == 0 {
				return written, nil
			}
		} else if ir.skipBytes > 0 {
			ir.skipBytes -= postfixSize
		}
	}
	if n, err = ir.w.Close(ir.skipBytes, maxbytes); err != nil {
		return n + written, err
	} else {
		maxbytes -= n
		written += n
	}
	return written, nil
}
