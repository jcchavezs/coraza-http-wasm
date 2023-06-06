.PHONY: build

build:
	mkdir -p build
	tinygo build -o build/coraza-http-wasm.wasm -scheduler=none --no-debug -target=wasi .

test:
	go test -v ./...