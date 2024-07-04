# coraza-http-wasm

Web Application Firewall WASM middleware built on top of Coraza and implementing the [http-wasm](https://http-wasm.io/) ABI.

## Getting started

`go run mage.go -l` lists all the available commands:

```bash
$ go run mage.go -l
Targets:
  build*    builds the wasm binary.
  e2e       runs e2e tests
  format    formats code in this repository.
  ftw       runs the FTW test suite
  lint      verifies code format.
  test      runs all unit tests.

* default target
```

### Building the binary

```bash
go run mage.go build
```

You will find the WASM plugin under `./build/coraza-http-wasm.wasm`.

### Basic Configuration

```json
{
  "directives": [
    "SecRuleEngine On",
    "SecDebugLog /dev/stdout",
    "SecDebugLogLevel 9",
    "SecRule REQUEST_URI \"@streq /admin\" \"id:101,phase:1,log,deny,status:403\""
  ]
}
```

### Test it

```console
curl -I 'http://localhost:8080/admin'    # 403
curl -I 'http://localhost:8080/anything' # 200
```

## Using CRS

One can leverage [CRS](https://github.com/coreruleset/coreruleset?tab=readme-ov-file#owasp-crs) by enabling it with `includeCRS` field and including the CRS rules:

```jsonc
{
  "includeCRS": true,
  "directives": [
    // mandatory as it initializes CRS:
    "Include @crs-setup.conf.example",
    // uses coraza recommended conf:
    "Include @coraza.conf-recommended",

    // override config with customs:
    "SecRuleEngine On"

    // loads CRS rules:
    "Include @owasp_crs/**.conf",
  ]
}
```

Notes:

1. [`@coraza.conf-recommended`](https://github.com/corazawaf/coraza-coreruleset/blob/main/rules/%40coraza.conf-recommended) comes with a set of defaults that have to be tweaked accordingly.
2. It is not mandatory to include all CRS rules, in fact it is discouraged to blindly include all rules in CRS. Instead one should select which ones are relevant for the environment and the ecosystem.
3. One can disable rules not used with [`SecRuleRemoveByID`](https://coraza.io/docs/seclang/directives/#secruleremovebyid) and [`SecRuleRemoveByTag`](https://coraza.io/docs/seclang/directives/#secruleremovebytag).
