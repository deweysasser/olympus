package poc_server

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOptions_createServer(t *testing.T) {
	o := &Options{}

	router, err := o.createServer()
	assert.NoError(t, err)

	server := httptest.NewServer(router)
	defer server.Close()

	r, e := http.Get(server.URL + "/status")
	assert.NoError(t, e)
	b, e := io.ReadAll(r.Body)
	assert.NoError(t, e)

	assert.Equal(t, "{\"status\":\"alive\"}", string(b))
	assert.Equal(t, 200, r.StatusCode)
}

func Benchmark_Status(b *testing.B) {
	o := &Options{}

	router, err := o.createServer()
	assert.NoError(b, err)

	server := httptest.NewServer(router)
	defer server.Close()

	for n := 0; n < b.N; n++ {

		r, e := http.Get(server.URL + "/status")
		assert.NoError(b, e)
		bytes, e := io.ReadAll(r.Body)
		assert.NoError(b, e)

		assert.Equal(b, "{\"status\":\"alive\"}", string(bytes))
		assert.Equal(b, 200, r.StatusCode)
	}
}
