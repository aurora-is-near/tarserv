package deliver

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Range_requests
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Range

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/aurora-is/tarserv/src/tarindex"
)

const defaultFilename = "data.tar"

type TarHandler struct {
	IndexDirectory string
}

func requestData(requestPath string) (index string) {
	if p := strings.LastIndex(requestPath, defaultFilename); p > 0 {
		requestPath = requestPath[0:p]
	}
	return path.Base(requestPath)
}

func (handler *TarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler.Handler(w, r)
}

func parseRange(r string) (start, end int64) {
	if pos := strings.Index(r, "="); pos < 0 {
		return 0, 0
	} else {
		r = r[pos+1:]
	}
	if pos := strings.Index(r, "-"); pos < 0 {
		return 0, 0
	} else {
		var start, stop int64
		bs, es := r[:pos], r[pos+1:]
		start, _ = strconv.ParseInt(bs, 10, 64)
		stop, _ = strconv.ParseInt(es, 10, 64)
		return start, stop
	}
}

func (handler *TarHandler) Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Accept-Ranges", "bytes")
	filename := r.URL.Query().Get("lastfile")
	startRange, endRange := parseRange(r.Header.Get("Range"))
	idxName := requestData(r.URL.Path)
	idxFile := path.Join(handler.IndexDirectory, fmt.Sprintf("%s.taridx", idxName))
	f, err := os.Open(idxFile)
	if err != nil {
		log.Printf("ERROR: Index %s: %s", idxName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer func() { _ = f.Close() }()
	postfixFile := &tarindex.PostfixFile{
		Name:    ".version",
		Content: []byte(idxName),
	}
	idxReader, err := tarindex.NewIndexReader(f, w, postfixFile)
	if err != nil {
		log.Printf("ERROR: Parse %s: %s", idxName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	setFunc := func(length int64) {
		w.Header().Add("Content-Type", "application/tar")
		w.Header().Add("Content-Disposition", "attachment; filename=\"data.tar\"")
		if startRange != 0 || endRange != 0 {
			if startRange >= idxReader.Size() {
				w.Header().Add("Content-Length", "0")
				w.Header().Add("Content-Range", fmt.Sprintf("bytes */%d", idxReader.Size()))
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			} else {
				if endRange == 0 {
					endRange = idxReader.Size()
				}
				w.Header().Add("Content-Length", strconv.FormatInt(endRange-startRange, 10))
				rangeHeader := fmt.Sprintf("bytes %d-%d/%d", startRange, endRange-1, idxReader.Size())
				w.Header().Add("Content-Range", rangeHeader)
				w.WriteHeader(http.StatusPartialContent)
			}
		}
	}
	if _, err := idxReader.SeekAndWrite(filename, startRange, endRange, setFunc); err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Printf("ERROR: Write %s (\"%s\", %d-%d): %s", idxName, filename, startRange, endRange, err)
		return
	}
}
