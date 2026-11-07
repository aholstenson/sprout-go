package otel_test

import (
	"net/http"
	"net/http/httptest"
	"sync"

	. "github.com/onsi/ginkgo/v2"
)

// otlpServer is an HTTP server that accepts OTLP requests and records the
// paths that were requested.
type otlpServer struct {
	*httptest.Server

	mu    sync.Mutex
	paths []string
}

// newOTLPServer starts an otlpServer. Ginkgo closes it after the spec.
func newOTLPServer() *otlpServer {
	s := &otlpServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.paths = append(s.paths, r.URL.Path)
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
	}))
	DeferCleanup(s.Close)
	return s
}

// Paths returns the paths that were requested.
func (s *otlpServer) Paths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.paths...)
}
