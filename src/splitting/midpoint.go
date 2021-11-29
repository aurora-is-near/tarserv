package splitting

import (
	"archive/tar"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

const blocksize int64 = 512

func tarPadding(size int64) int64 {
	if size%blocksize == 0 {
		return 0
	}
	return blocksize - size%blocksize
}

type PosReader struct {
	prevPos int64
	pos     int64
	r       io.Reader
}

func NewPosReader(r io.Reader) *PosReader {
	return &PosReader{r: r}
}
func (pr *PosReader) Read(p []byte) (n int, err error) {
	n, err = pr.r.Read(p)
	pr.prevPos = pr.pos
	pr.pos = pr.pos + int64(n)
	return n, err
}

func midpoint2(filename string) (lastbyte int64, err error) {
	var header *tar.Header
	var stop int64
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return 0, err
	}
	stop = stat.Size() / 2
	pr := NewPosReader(f)
	tr := tar.NewReader(pr)
	for header, err = tr.Next(); err == nil; header, err = tr.Next() {
		if pr.pos >= stop {
			return pr.pos + header.Size + tarPadding(header.Size), nil
		}
	}
	return 0, io.ErrShortBuffer
}

func splitfile(filename string, midpoint int64) error {
	destName := filename + ".part2"
	destF, err := os.Create(destName)
	if err != nil {
		return err
	}
	defer destF.Close()
	sourceF, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer sourceF.Close()
	pos, err := sourceF.Seek(midpoint, io.SeekStart)
	if err != nil {
		return err
	}
	if pos != midpoint {
		panic("Seek failure")
	}
	if _, err = io.Copy(destF, sourceF); err != nil {
		return err
	}
	return os.Truncate(filename, midpoint)
}

// SplitTarMiddle splits a tarfile roughly at it's middle, preserving headers so that each part is a valid tar file.
// It truncates the input tarfile in place, and copies the remainder into a file called "<tarfile>.part2".
func SplitTarMiddle(tarfile string) error {
	mid, err := midpoint2(tarfile)
	if err != nil {
		return err
	}
	return splitfile(tarfile, mid)
}

func ReadSHA256(tarfile string, w io.Writer) error {
	var header *tar.Header
	f, err := os.Open(tarfile)
	if err != nil {
		return err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	for header, err = tr.Next(); err == nil; header, err = tr.Next() {
		if header.Typeflag != tar.TypeReg {
			continue
		}
		h := sha256.New()
		if _, err := io.Copy(h, tr); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%x  %s\n", h.Sum(nil), header.Name); err != nil {
			return err
		}
	}
	return nil
}
