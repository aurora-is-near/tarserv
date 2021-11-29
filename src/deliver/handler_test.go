package deliver

import (
	"net/http"
	"testing"
	"time"
)

func TestHandler(t *testing.T) {
	h := &TarHandler{
		IndexDirectory: "/tmp",
	}
	prefix := "/something/"
	address := "127.0.0.1:8081"
	mux := http.NewServeMux()
	mux.Handle(prefix, http.StripPrefix(prefix, h))
	_ = http.ListenAndServe(address, mux)
	time.Sleep(time.Hour)
}
