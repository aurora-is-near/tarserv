package tarserv

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path"
	"time"
)

type byteSeekOption struct {
	filename  string
	skipbytes int64
}

func (opt byteSeekOption) applyOption(option *activeConfig) {
	option.startFile = opt.filename
	option.mockwrite = true
	option.skipOutBytes = opt.skipbytes
}

//goland:noinspection GoUnusedExportedFunction
func OptByteSeek(firstFile string, skipBytes int64) Option {
	return &byteSeekOption{
		filename:  firstFile,
		skipbytes: skipBytes,
	}
}

type rebaseOption struct {
	dir string
}

func (opt rebaseOption) applyOption(option *activeConfig) {
	option.PathType = rebase(opt.dir)
}

// OptRebase returns an Option that rebases the tar entries to dir.
func OptRebase(dir string) Option {
	return &rebaseOption{dir: dir}
}

func rebase(dir string) PathFunc {
	if dir[len(dir)-1] == '/' {
		dir = dir[:len(dir)-1]
	}
	return func(base string) PathRewriteFunc {
		l := len(base)
		return func(d string) string {
			if len(d) == l {
				return dir
			}
			return dir + d[l:]
		}
	}
}

// OptRelative will rebase the tar file to relative paths.
var OptRelative = new(optRelative)

type optRelative struct{}

func (opt optRelative) applyOption(option *activeConfig) {
	option.PathType = relativePath
}

func relativePath(base string) PathRewriteFunc {
	l := len(base)
	return func(d string) string {
		if len(d) == l {
			return "./"
		}
		return "." + d[l:]
	}
}

// OptAbsolute will rebase the tar file for absolute original paths.
//goland:noinspection GoUnusedGlobalVariable
var OptAbsolute = new(optAbsolute)

type optAbsolute struct{}

func (opt optAbsolute) applyOption(option *activeConfig) {
	option.PathType = absolutePath
}

// absolutePath will rebase the tar file for absolute original paths.
func absolutePath(base string) PathRewriteFunc {
	_ = base
	return func(d string) string { return d }
}

type setUIDOption struct {
	uid int
}

func (opt setUIDOption) applyOption(option *activeConfig) {
	option.HeaderFixes = append(option.HeaderFixes,
		func(header *tar.Header) {
			header.Uid = opt.uid
			header.Uname = ""
		})
}

// OptUID sets all file user IDs to uid.
//goland:noinspection GoExportedFuncWithUnexportedType
func OptUID(uid int) setUIDOption {
	return setUIDOption{uid: uid}
}

// OptNumericIDs sets all IDs to numeric.
var OptNumericIDs = new(optNumericIDs)

type optNumericIDs struct{}

func (opt optNumericIDs) applyOption(option *activeConfig) {
	option.HeaderFixes = append(option.HeaderFixes,
		func(header *tar.Header) {
			header.Uname = ""
			header.Gname = ""
		})
}

type setGIDOption struct {
	gid int
}

func (opt setGIDOption) applyOption(option *activeConfig) {
	option.HeaderFixes = append(option.HeaderFixes,
		func(header *tar.Header) {
			header.Gid = opt.gid
			header.Gname = ""
		})
}

// OptGID sets all file group IDs to uid.
//goland:noinspection GoExportedFuncWithUnexportedType
func OptGID(gid int) setGIDOption {
	return setGIDOption{gid: gid}
}

type appendFileOpt struct {
	name string
	r    io.Reader
	mode os.FileMode
}

func (opt appendFileOpt) applyOption(option *activeConfig) {
	option.Appends = append(option.Appends, opt)
}

func (opt appendFileOpt) Append(dir string, w *tar.Writer, options activeConfig) error {
	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, opt.r)
	if err != nil {
		return err
	}
	now := time.Now()
	header := &tar.Header{
		Typeflag:   tar.TypeReg,
		Size:       n,
		Mode:       int64(opt.mode),
		ModTime:    now,
		AccessTime: now,
		ChangeTime: now,
	}
	header.Name = options.ActivePathType(path.Join(dir, opt.name))
	fixHeader(header, options)
	if err := w.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(w, buf)
	return err
}

// OptAppendFile appends a file to the tar with the given name and content read from r.
//goland:noinspection GoExportedFuncWithUnexportedType
func OptAppendFile(name string, mode os.FileMode, r io.Reader) appendFileOpt {
	return appendFileOpt{name: name, r: r, mode: mode}
}

// Option is an option for tarfile creation.
type Option interface {
	applyOption(option *activeConfig)
}

type headerFixFunc func(header *tar.Header)

func newOptions() *activeConfig {
	return &activeConfig{
		PathType:    relativePath,
		HeaderFixes: make([]headerFixFunc, 0, 4),
	}
}
