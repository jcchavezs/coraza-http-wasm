package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"

	"github.com/http-wasm/http-wasm-host-go/handler"
	wasm "github.com/http-wasm/http-wasm-host-go/handler/nethttp"
)

//go:embed build/coraza-http-wasm.wasm
var guest string

func exampleHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Hello world, transaction not disrupted."))
}

func ExampleMain() {
	ctx := context.Background()

	h, err := wasm.NewMiddleware(
		ctx,
		[]byte(guest),
		handler.GuestConfig([]byte(`
		{
			"directives": [
				"SecRuleEngine On",
				"SecDebugLogLevel 9",
				"SecDebugLog /dev/stdout",
				"SecRule REQUEST_URI \"@rx .\" \"phase:1,deny,status:403,id:'1234'\""
			]
		}`),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create middleware: %s", err.Error())
	}
	defer h.Close(ctx)

	w := h.NewHandler(ctx, http.HandlerFunc(exampleHandler))

	srvAddress := ":8080"
	srv := &http.Server{Addr: srvAddress, Handler: w}

	go srv.ListenAndServe()

	defer srv.Close()

	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost%s?key=<alert>", srvAddress), nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Failed to call the server: %s", err.Error())
	}

	fmt.Println(res.StatusCode)
	res.Body.Close()

	// Output: 403
}
