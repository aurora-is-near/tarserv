package tarindex

import (
	"archive/tar"
	"bytes"
	"encoding/binary"
	"io/fs"
	"time"
)

func paddedTarBlockSize(size int64) int64 {
	if size%tarBlockSize == 0 {
		return size
	}
	return tarBlockSize + (size/tarBlockSize)*tarBlockSize
}

func (entry *ListEntry) TarSize() int64 {
	switch entry.Type {
	case EntryTypeLink:
		return tarHeaderSize
	case EntryTypeDirectory:
		return tarHeaderSize
	case EntryTypeFile:
		return tarHeaderSize + paddedTarBlockSize(entry.Size)
	default:
		return 0
	}
}

// BinaryEntry contains the size or offset, type and path of an entry.
type BinaryEntry [binaryEntrySize]byte

// BinaryEntry returns the binary entry for the ListEntry. If offset is given, it will be added to the size.
// This allows quick calculation about the last byte in a tar file occupied by the given entry.
func (entry *ListEntry) BinaryEntry(offset int64) (newOffset int64, binEntry *BinaryEntry) {
	bin := new(BinaryEntry)
	size := entry.TarSize() + offset
	writeSize(bin, size)
	bin[binaryTypePos] = byte(entry.Type)
	copy(bin[binaryNamePos:binaryNameEnd], entry.Name)
	return size, bin
}

func writeSize(d *BinaryEntry, size int64) {
	binary.LittleEndian.PutUint64(d[binarySizePos:binarySizeEnd], uint64(size))
}

func readSize(d BinaryEntry) int64 {
	return int64(binary.LittleEndian.Uint64(d[binarySizePos:binarySizeEnd]))
}

func readName(d BinaryEntry) string {
	return string(bytes.TrimRightFunc(d[binaryNamePos:binaryNameEnd], func(r rune) bool { return r == 0x00 }))
}

// ToListEntry returns a list entry from a BinaryEntry.
func (bin BinaryEntry) ToListEntry(offset int64) *ListEntry {
	size := readSize(bin)
	name := readName(bin)
	return &ListEntry{
		Name:      name,
		Type:      EntryType(bin[binaryTypePos]),
		Size:      size - offset,
		FirstByte: offset,
		LastByte:  size,
	}
}

// tarHeaderBytesFromFileInfo creates a tar header of fi.
func tarHeaderBytesFromFileInfo(entry *ListEntry, fi fs.FileInfo, link string, fixHeader func(hdr *tar.Header)) ([]byte, error) {
	hdr, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return nil, err
	}
	hdr.Name = entry.Name
	return tarHeaderBytes(hdr, fixHeader)
}

func tarHeaderBytes(hdr *tar.Header, fixHeader func(hdr *tar.Header)) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := tar.NewWriter(buf)
	hdr.Format = tarHeaderFormat
	if fixHeader != nil {
		fixHeader(hdr)
	}
	hdr.ChangeTime = time.Time{}
	hdr.AccessTime = time.Time{}
	hdr.PAXRecords = nil
	if err := w.WriteHeader(hdr); err != nil {
		return nil, err
	}
	_ = w.Flush()
	return buf.Bytes(), nil
}
