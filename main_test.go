//go:build e2e
// +build e2e

package main_test

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/http-wasm/http-wasm-host-go/handler"
	nethttp "github.com/http-wasm/http-wasm-host-go/handler/nethttp"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

// testCtx is an arbitrary, non-default context. Non-nil also prevents linter errors.
var testCtx = context.WithValue(context.Background(), struct{}{}, "arbitrary")

//go:embed build/coraza-http-wasm.wasm
var guest []byte

func TestE2E(t *testing.T) {
	// TODO: replace this tests with coraza http tests
	var stdoutBuf, stderrBuf bytes.Buffer
	moduleConfig := wazero.NewModuleConfig().WithStdout(&stdoutBuf).WithStderr(&stderrBuf)

	// Configure and compile the WebAssembly guest binary.
	mw, err := nethttp.NewMiddleware(testCtx, guest,
		handler.ModuleConfig(moduleConfig),
		handler.GuestConfig([]byte(
			`
			SecRuleEngine On
			SecDebugLogLevel 9
			SecRule REQUEST_METHOD "@streq GET" "id:1,phase:1,deny,status:403"
			`,
		)),
	)
	if err != nil {
		t.Error(err)
	}
	defer mw.Close(testCtx)

	// Wrap the test handler with one implemented in WebAssembly.
	wrapped := mw.NewHandler(testCtx, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("hello world"))
	}))
	require.NoError(t, err)

	// Start the server with the wrapped handler.
	ts := httptest.NewServer(wrapped)
	defer ts.Close()

	t.Run("GET is denied", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
		// Make a client request and invoke the test.
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		content, _ := io.ReadAll(resp.Body)
		require.Empty(t, content)
		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("POST is allowed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL, nil)
		// Make a client request and invoke the test.
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		content, _ := io.ReadAll(resp.Body)
		require.Equal(t, "hello world", string(content))
		require.Equal(t, http.StatusAccepted, resp.StatusCode)
	})
}
