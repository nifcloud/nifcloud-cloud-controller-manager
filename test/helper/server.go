package helper

import (
	"net/http"
	"net/http/httptest"

	"github.com/onsi/ginkgo/v2"
)

func NewTestServer(handler http.Handler) *httptest.Server {
	return httptest.NewServer(panicWrapper(handler))
}

func panicWrapper(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer ginkgo.GinkgoRecover()
		handler.ServeHTTP(w, r)
	})
}
