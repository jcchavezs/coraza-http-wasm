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
