//go:build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/sh"
)

var Default = Build

var envoyBinaryPath = "./envoy/envoybin/envoy"

// Build builds the Coraza wasm plugin.
func Build() error {
	if err := os.MkdirAll("build", 0755); err != nil {
		return err
	}
	return sh.RunV("tinygo", "build", "-gc=custom", "-tags=custommalloc", "-opt=2", "-o", filepath.Join("build", "coraza-http-wasm.wasm"), "-scheduler=none", "--no-debug", "-target=wasi")
}

// Test runs all unit tests.
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// E2e runs e2e tests with wazero
func E2e() error {
	return sh.RunV("go", "test", "-tags=e2e", "-v", "./...")
}

func checkEnvoyBinary() error {
	_, err := os.Stat(envoyBinaryPath)
	if err != nil {
		return fmt.Errorf("envoy binary not found at %s", envoyBinaryPath)
	}
	return nil
}

// EnvoyE2e runs e2e tests against Envoy with the coraza-http-wasm plugin. Requires docker-compose.
func EnvoyE2e() error {
	var err error

	if err = checkEnvoyBinary(); err != nil {
		return err
	}

	if err = sh.RunV("docker-compose", "--file", "envoy/e2e/docker-compose.yml", "up", "-d", "envoy"); err != nil {
		return err
	}
	defer func() {
		_ = sh.RunV("docker-compose", "--file", "envoy/e2e/docker-compose.yml", "down", "-v")
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
		sh.RunV("docker-compose", "-f", "envoy/e2e/docker-compose.yml", "logs", "envoy")
	}
	return err
}

// EnvoyFtw runs ftw tests against Envoy with the coraza-http-wasm plugin. Requires docker-compose.
func EnvoyFtw() error {

	if err := checkEnvoyBinary(); err != nil {
		return err
	}

	if err := sh.RunV("docker-compose", "--file", "envoy/ftw/docker-compose.yml", "build", "--pull"); err != nil {
		return err
	}
	defer func() {
		_ = sh.RunV("docker-compose", "--file", "envoy/ftw/docker-compose.yml", "down", "-v")
	}()
	env := map[string]string{
		"FTW_CLOUDMODE": os.Getenv("FTW_CLOUDMODE"),
		"FTW_INCLUDE":   os.Getenv("FTW_INCLUDE"),
	}
	task := "ftw"
	if os.Getenv("MEMSTATS") == "true" {
		task = "ftw-memstats"
	}
	return sh.RunWithV(env, "docker-compose", "--file", "envoy/ftw/docker-compose.yml", "run", "--rm", task)
}

// RunExample spins up the test environment loading Envoy with the coraza-http-wasm plugin, access at http://localhost:8080. Requires docker-compose.
func RunExample() error {
	if err := checkEnvoyBinary(); err != nil {
		return err
	}
	return sh.RunV("docker-compose", "--file", "envoy/example/docker-compose.yml", "up", "-d", "envoy-logs")
}

// TeardownExample tears down the test environment. Requires docker-compose.
func TeardownExample() error {
	if err := checkEnvoyBinary(); err != nil {
		return err
	}
	return sh.RunV("docker-compose", "--file", "envoy/example/docker-compose.yml", "down")
}

// ReloadExample reload the test environment (container) in case of envoy or wasm update. Requires docker-compose
func ReloadExample() error {
	if err := checkEnvoyBinary(); err != nil {
		return err
	}
	return sh.RunV("docker-compose", "--file", "envoy/example/docker-compose.yml", "restart")
}
