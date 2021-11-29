package tarserv

import (
	"testing"
	"time"
)

func TestHTTP(t *testing.T) {
	go func() { _ = Serve("127.0.0.1:8080", "/db/", "/tmp", ".version") }()
	time.Sleep(time.Second * 10)
}
