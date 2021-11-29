package tarserv

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"path"
)

// TarHandler is a http.Handler that serves a sub-directory of SourceDir (only last
type TarHandler struct {
	SourceDir      string
	AppendFileName string
}

func (handler *TarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler.Handler(w, r)
}

func (handler *TarHandler) Handler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) == 0 {
		w.WriteHeader(403)
		return
	}
	dir := path.Join(handler.SourceDir, r.URL.Path)
	stat, err := os.Stat(dir)
	if err != nil || !stat.IsDir() {
		w.WriteHeader(404)
		return
	}
	buf := new(bytes.Buffer)
	buf.Write([]byte(dir))
	opts := []Option{
		OptAppendFile(handler.AppendFileName, 0600, buf),
		OptRelative,
		OptNumericIDs,
		OptGID(0),
		OptUID(0),
	}
	w.Header().Add("Content-Type", "application/tar")
	w.Header().Add("Content-Disposition", "inline; filename=\"data.tar\"")
	if err := NewTar(dir, w, opts...); err != nil {
		log.Printf("Error creating tar: %s", err)
	}
}

func Serve(address, prefix, sourceDir, appendFilename string) error {
	db := &TarHandler{
		SourceDir:      sourceDir,
		AppendFileName: appendFilename,
	}
	mux := http.NewServeMux()
	mux.Handle(prefix, http.StripPrefix(prefix, db))
	return http.ListenAndServe(address, mux)
}
