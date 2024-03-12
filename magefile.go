//go:build mage

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Build

var (
	golangCILintVer = "v1.56.2" // https://github.com/golangci/golangci-lint/releases
	gosImportsVer   = "v0.3.8"  // https://github.com/rinchsan/gosimports/releases/tag/v0.3.1
)

var errCommitFormatting = errors.New("files not formatted, please commit formatting changes")

// Format formats code in this repository.
func Format() error {
	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return err
	}

	return sh.RunV("go", "run", fmt.Sprintf("github.com/rinchsan/gosimports/cmd/gosimports@%s", gosImportsVer),
		"-w",
		"-local",
		"github.com/jcchavezs/coraza-http-wasm",
		".")
}

// Lint verifies code quality.
func Lint() error {
	if err := sh.RunV("go", "run", fmt.Sprintf("github.com/golangci/golangci-lint/cmd/golangci-lint@%s", golangCILintVer), "run"); err != nil {
		return err
	}

	mg.SerialDeps(Format)

	if sh.Run("git", "diff", "--exit-code") != nil {
		return errCommitFormatting
	}

	return nil
}

// Build builds the Coraza wasm plugin.
func Build() error {
	if err := os.MkdirAll("build", 0755); err != nil {
		return err
	}

	return sh.RunV("tinygo", "build", "-o", filepath.Join("build", "coraza-http-wasm.wasm"), "-gc=custom", "-tags=custommalloc", "-scheduler=none", "--no-debug", "-target=wasi")
}

// Test runs all unit tests.
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// E2e runs e2e tests with wazero
func E2e() error {
	return sh.RunV("go", "test", "-run=^TestE2E", "-tags=e2e", "-v", ".")
}

func copy(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file: %v", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func FTW() error {
	var (
		binSrc = filepath.Join("build", "coraza-http-wasm.wasm")
		binDst = filepath.Join("testing", "coreruleset", "build", "coraza-http-wasm.wasm")
	)
	if err := copy(binSrc, binDst); err != nil {
		return fmt.Errorf("copying build: %v", err)
	}
	defer os.Remove(binDst)

	return sh.RunV("go", "test", "./testing/coreruleset")
}
