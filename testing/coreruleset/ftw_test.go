package coreruleset

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"io/fs"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coreruleset/go-ftw/config"
	"github.com/coreruleset/go-ftw/output"
	"github.com/coreruleset/go-ftw/runner"
	"github.com/mccutchen/go-httpbin/v2/httpbin"

	"github.com/bmatcuk/doublestar/v4"
	crstests "github.com/corazawaf/coraza-coreruleset/v4/tests"

	"github.com/coreruleset/go-ftw/test"
	"github.com/http-wasm/http-wasm-host-go/api"
	"github.com/http-wasm/http-wasm-host-go/handler"
	wasm "github.com/http-wasm/http-wasm-host-go/handler/nethttp"
	"github.com/rs/zerolog"
)

type logger struct {
	t *testing.T
	f *bufio.Writer
}

func (logger) IsEnabled(api.LogLevel) bool { return true }

func (l logger) Log(_ context.Context, _ api.LogLevel, msg string) {
	if strings.Contains(msg, "Coraza: Warning.") {
		l.f.Write([]byte(msg + "\n"))
		l.f.Flush()
		return
	}

	//l.t.Log(msg)
}

//go:embed build/coraza-http-wasm.wasm
var guestWasm []byte

func TestFTW(t *testing.T) {
	const directives = `
# Coraza config
Include @coraza.conf-recommended

# Custom Rules for testing and eventually overrides of the basic Coraza config
SecResponseBodyMimeType text/plain",
SecDefaultAction "phase:3,log,auditlog,pass"
SecDefaultAction "phase:4,log,auditlog,pass"
SecDefaultAction "phase:5,log,auditlog,pass"

# Rule 900005 from https://github.com/coreruleset/coreruleset/blob/v4.0/dev/tests/regression/README.md#requirements
SecAction "id:900005,\
  phase:1,\
  nolog,\
  pass,\
  ctl:ruleEngine=DetectionOnly,\
  ctl:ruleRemoveById=910000,\
  setvar:tx.blocking_paranoia_level=4,\
  setvar:tx.crs_validate_utf8_encoding=1,\
  setvar:tx.arg_name_length=100,\
  setvar:tx.arg_length=400,\
  setvar:tx.total_arg_length=64000,\
  setvar:tx.max_num_args=255,\
  setvar:tx.max_file_size=64100,\
  setvar:tx.combined_file_sizes=65535"

# Write the value from the X-CRS-Test header as a marker to the log
# Requests with X-CRS-Test header will not be matched by any rule. See https://github.com/coreruleset/go-ftw/pull/133
SecRule REQUEST_HEADERS:X-CRS-Test "@rx ^.*$" \
  "id:999999,\
  phase:1,\
  pass,\
  t:none,\
  log,\
  msg:'X-CRS-Test %{MATCHED_VAR}',\
  ctl:ruleRemoveById=1-999999"

# CRS basic config
Include @crs-setup.conf.example

# CRS rules (on top of which are applied the previously defined SecDefaultAction)
Include @owasp_crs/*.conf
`

	errorPath := filepath.Join(t.TempDir(), "error.log")
	errorFile, err := os.Create(errorPath)
	if err != nil {
		t.Fatalf("failed to create error log: %v", err)
	}

	mw, err := wasm.NewMiddleware(
		context.Background(),
		guestWasm,
		handler.GuestConfig([]byte(fmt.Sprintf("{\"directives\": [ %q ]}", directives))),
		handler.Logger(logger{t, bufio.NewWriter(errorFile)}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer mw.Close(context.Background())

	var tests []*test.FTWTest
	err = doublestar.GlobWalk(crstests.FS, "**/*.yaml", func(path string, d os.DirEntry) error {
		yaml, err := fs.ReadFile(crstests.FS, path)
		if err != nil {
			return err
		}
		ftwt, err := test.GetTestFromYaml(yaml)
		if err != nil {
			return err
		}
		tests = append(tests, ftwt)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(tests) == 0 {
		t.Fatal("no tests found")
	}

	s := httptest.NewServer(mw.NewHandler(context.Background(), httpbin.New().Handler()))
	defer s.Close()

	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	// TODO(anuraaga): Don't use global config for FTW for better support of programmatic.
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	cfg, err := config.NewConfigFromFile(".ftw.yml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.WithLogfile(errorPath)
	cfg.TestOverride.Overrides.DestAddr = &host
	cfg.TestOverride.Overrides.Port = &port

	res, err := runner.Run(cfg, tests, runner.RunnerConfig{
		ShowTime:    false,
		ReadTimeout: 10 * time.Second,
	}, output.NewOutput("quiet", os.Stdout))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Stats.Failed) > 0 {
		t.Errorf("failed tests: %v", res.Stats.Failed)
	}
}
