# coraza-http-wasm

Web Application Firewall WASM filter built on top of Coraza and implementing the [http-wasm](https://http-wasm.io/) ABI.

## Getting started

`go run mage.go -l` lists all the available commands:

```bash
$ go run mage.go -l
Targets:
  build*             builds the Coraza wasm plugin.
  e2e                runs e2e tests with wazero
  envoyE2e           runs e2e tests against Envoy with the coraza-http-wasm plugin.
  envoyFtw           runs ftw tests against Envoy with the coraza-http-wasm plugin.
  reloadExample      reload the test environment (container) in case of envoy or wasm update.
  runExample         spins up the test environment loading Envoy with the coraza-http-wasm plugin, access at http://localhost:8080.
  teardownExample    tears down the test environment.
  test               runs all unit tests.

* default target
```

**Note**: In order to run Envoy specific mage commands, an Envoy binary that supports http-wasm filter is needed. Add it to `./envoy/envoybin` naming it `envoy`.

### Building the filter

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

```
curl -I 'http://localhost:8080/admin'    # 403
curl -I 'http://localhost:8080/anything' # 200
```
