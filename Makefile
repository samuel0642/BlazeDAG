.PHONY: build test

build:
	go build -o blazedag ./cmd/blazedag

test: build
	./scripts/test_timeouts.sh

clean:
	rm -f blazedag
	rm -rf /tmp/blazedag_test_* 