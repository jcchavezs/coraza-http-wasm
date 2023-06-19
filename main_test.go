//go:build e2e
// +build e2e

package main_test

import (
	"bytes"
	"context"
	_ "embed"
	"net/http"
	"net/http/httptest"
	"testing"

	e2e "github.com/corazawaf/coraza/v3/http/e2e/pkg"
	"github.com/http-wasm/http-wasm-host-go/handler"
	nethttp "github.com/http-wasm/http-wasm-host-go/handler/nethttp"
	"github.com/mccutchen/go-httpbin/v2/httpbin"
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
		handler.GuestConfig([]byte(`
		SecRuleEngine On
		SecResponseBodyAccess On
		SecResponseBodyMimeType application/json
		# Custom rule for Coraza config check (ensuring that these configs are used)
		SecRule &REQUEST_HEADERS:coraza-e2e "@eq 0" "id:100,phase:1,deny,status:424,log,msg:'Coraza E2E - Missing header'"
		# Custom rules for e2e testing
		SecRule REQUEST_URI "@streq /admin" "id:101,phase:1,t:lowercase,log,deny"
		SecRule REQUEST_BODY "@rx maliciouspayload" "id:102,phase:2,t:lowercase,log,deny"
		SecRule RESPONSE_HEADERS:pass "@rx leak" "id:103,phase:3,t:lowercase,log,deny"
		SecRule RESPONSE_BODY "@contains responsebodycode" "id:104,phase:4,t:lowercase,log,deny"
		# Custom rules mimicking the following CRS rules: 941100, 942100, 913100
		SecRule ARGS_NAMES|ARGS "@detectXSS" "id:9411,phase:2,t:none,t:utf8toUnicode,t:urlDecodeUni,t:htmlEntityDecode,t:jsDecode,t:cssDecode,t:removeNulls,log,deny"
		SecRule ARGS_NAMES|ARGS "@detectSQLi" "id:9421,phase:2,t:none,t:utf8toUnicode,t:urlDecodeUni,t:removeNulls,multiMatch,log,deny"
		SecRule REQUEST_HEADERS:User-Agent "@pm grabber masscan" "id:9131,phase:1,t:none,log,deny"
		SecRequestBodyAccess On
		`,
		)),
	)
	if err != nil {
		t.Error(err)
	}
	defer mw.Close(testCtx)

	httpbin := httpbin.New()

	// Wrap the test handler with one implemented in WebAssembly.
	wrapped := mw.NewHandler(testCtx, httpbin)
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.Handle("/status/200", httpbin) // Health check
	mux.Handle("/", wrapped)

	// Create the server with the WAF and the reverse proxy.
	ts := httptest.NewServer(mux)
	defer ts.Close()

	err = e2e.Run(e2e.Config{
		ProxiedEntrypoint: ts.URL,
		HttpbinEntrypoint: ts.URL,
	})
	require.NoError(t, err)
}
