//go:build mage

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/magefile/mage/sh"
)

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func E2e() error {
	var err error

	if err = os.Mkdir("build", 0755); err != nil {
		return err
	}

	_, err = copy("../../build/coraza-http-wasm.wasm", "./build/coraza-http-wasm.wasm")
	if err != nil {
		return err
	}
	defer os.Remove("./build")

	if err = sh.RunV("docker-compose", "--file", "docker-compose.yml", "up", "-d", "traefik"); err != nil {
		return err
	}
	defer func() {
		_ = sh.RunV("docker-compose", "--file", "docker-compose.yml", "down", "-v")
	}()

	envoyHost := os.Getenv("ENVOY_HOST")
	if envoyHost == "" {
		envoyHost = "localhost:8080"
	}
	httpbinHost := os.Getenv("HTTPBIN_HOST")
	if httpbinHost == "" {
		httpbinHost = "localhost:8000"
	}

	if err = sh.RunV("go", "run", "github.com/corazawaf/coraza/v3/http/e2e/cmd/httpe2e@main", "--proxy-hostport",
		"http://"+envoyHost, "--httpbin-hostport", "http://"+httpbinHost); err != nil {
		sh.RunV("docker-compose", "-f", "docker-compose.yml", "logs", "traefik")
	}

	return err
}
