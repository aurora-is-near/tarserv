package tarindex

import "archive/tar"

const (
	tarHeaderFormat = tar.FormatUSTAR

	tarHeaderSize int64 = 512
	tarBlockSize  int64 = 512
	tarFooterSize       = tarBlockSize * 2

	binarySizeLen = 8
	binaryTypeLen = 1
	binaryNameLen = 256
	binarySizePos = 0
	binarySizeEnd = binarySizePos + binarySizeLen
	binaryTypePos = binarySizeEnd
	binaryTypeEnd = binaryTypePos + binaryTypeLen
	binaryNamePos = binaryTypeEnd
	binaryNameEnd = binaryNamePos + binaryNameLen

	binaryEntrySize int = binarySizeLen + binaryTypeLen + binaryNameLen
)

type block [tarBlockSize]byte

var zeroBlock block

type EntryType byte

const (
	EntryTypeHeader    EntryType = 0xff
	EntryTypeDirectory EntryType = 0x01
	EntryTypeFile      EntryType = 0x02
	EntryTypeLink      EntryType = 0x03
)

// ListEntry describes an entry in a list of tar file entries.
type ListEntry struct {
	Size      int64     // Size of the entry.
	Name      string    // Path of filesystem object.
	Type      EntryType // Directory, link, or regular file.
	FirstByte int64     // First byte occupied in the tar file. Only populated when reading.
	LastByte  int64     // Last byte occupied in the tar file. Only populated when reading.
}
