# coraza-http-wasm

Web Application Firewall WASM filter built on top of Coraza and implementing the http-wasm ABI.

## Getting started

```bash
# Build the middleware in build/coraza-http-wasm.wasm:
$ make build

# Run the example:
$ go test -timeout 300s -run ^ExampleMain$ github.com/corazawaf/coraza-http-wasm
```

## Configuration

```json
{
   "directives": [
    "SecRuleEngine On",
    "SecDebugLog /dev/stdout",
    "SecDebugLogLevel 9",
   ]
  }
```
