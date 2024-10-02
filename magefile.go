//go:build mage

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/tetratelabs/wabin/binary"
	"github.com/tetratelabs/wabin/wasm"
)

var Default = Build

var (
	minGoVersion     = "1.22"
	minTinygoVersion = "0.33.0"
	golangCILintVer  = "v1.61.0" // https://github.com/golangci/golangci-lint/releases
	gosImportsVer    = "v0.3.8"  // https://github.com/rinchsan/gosimports/releases/tag/v0.3.1
)

var errCommitFormatting = errors.New("files not formatted, please commit formatting changes")

func init() {
	for _, check := range []struct {
		lang       string
		minVersion string
	}{
		{"tinygo", minTinygoVersion},
		{"go", minGoVersion},
	} {
		if err := checkVersion(check.lang, check.minVersion); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// checkVersion checks the minimum version of the specified language is supported.
// Note: While it is likely, there are no guarantees that a newer version of the language will work
func checkVersion(lang string, minVersion string) error {
	var compare []string

	switch lang {
	case "go":
		// Version can/cannot include patch version e.g.
		// - go version go1.19 darwin/arm64
		// - go version go1.19.2 darwin/amd64
		goVersionRegex := regexp.MustCompile("go([0-9]+).([0-9]+).?([0-9]+)?")
		v, err := sh.Output("go", "version")
		if err != nil {
			return fmt.Errorf("unexpected go error: %v", err)
		}
		compare = goVersionRegex.FindStringSubmatch(v)
		if len(compare) != 4 {
			return fmt.Errorf("unexpected go semver: %q", v)
		}
	case "tinygo":
		tinygoVersionRegex := regexp.MustCompile("tinygo version ([0-9]+).([0-9]+).?([0-9]+)?")
		v, err := sh.Output("tinygo", "version")
		if err != nil {
			return fmt.Errorf("unexpected tinygo error: %v", err)
		}
		// Assume a dev build is valid.
		if strings.Contains(v, "-dev") {
			return nil
		}
		compare = tinygoVersionRegex.FindStringSubmatch(v)
		if len(compare) != 4 {
			return fmt.Errorf("unexpected tinygo semver: %q", v)
		}
	default:
		return fmt.Errorf("unexpected language: %s", lang)
	}

	compare = compare[1:]
	if compare[2] == "" {
		compare[2] = "0"
	}

	base := strings.SplitN(minVersion, ".", 3)
	if len(base) == 2 {
		base = append(base, "0")
	}
	for i := 0; i < 3; i++ {
		baseN, _ := strconv.Atoi(base[i])
		compareN, _ := strconv.Atoi(compare[i])
		if baseN > compareN {
			return fmt.Errorf("unexpected %s version, minimum want %q, have %q", lang, minVersion, strings.Join(compare, "."))
		}
	}
	return nil
}

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

// Lint verifies code format.
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

// Build builds the wasm binary.
func Build() error {
	if err := os.MkdirAll("build", 0755); err != nil {
		return err
	}

	err := sh.RunV("tinygo", "build", "-o", filepath.Join("build", "coraza-http-wasm-raw.wasm"), "-opt=2", "-gc=custom", "-tags='custommalloc no_fs_access'", "-scheduler=none", "--no-debug", "-target=wasi")
	if err != nil {
		return err
	}

	return patchWasm(filepath.Join("build", "coraza-http-wasm-raw.wasm"), filepath.Join("build", "coraza-http-wasm.wasm"), 1050)
}

func patchWasm(inPath, outPath string, initialPages int) error {
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}
	mod, err := binary.DecodeModule(raw, wasm.CoreFeaturesV2)
	if err != nil {
		return err
	}

	mod.MemorySection.Min = uint32(initialPages)

	out := binary.EncodeModule(mod)
	if err = os.WriteFile(outPath, out, 0644); err != nil {
		return err
	}

	return nil
}

// Test runs all unit tests.
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// E2e runs e2e tests
func E2e() error {
	return sh.RunV("go", "test", "-count=1", "-run=^TestE2E", "-tags=e2e", "-v", ".")
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

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destination, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file: %v", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// FTW runs the FTW test suite
func FTW() error {
	var (
		binSrc = filepath.Join("build", "coraza-http-wasm.wasm")
		binDst = filepath.Join("testing", "coreruleset", "build", "coraza-http-wasm.wasm")
	)

	if err := copy(binSrc, binDst); err != nil {
		return fmt.Errorf("copying build: %v", err)
	}
	defer os.Remove(binDst)

	return sh.RunV("go", "test", "-count=1", "./testing/coreruleset")
}
