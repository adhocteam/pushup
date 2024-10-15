all: pushup

.PHONY: pushup clean install test

pushup:
	go build -o pushup ./cmd/pushup

clean:
	rm -f pushup

install:
	go install -v ./cmd/...

test:
	go test ./...

internal/parser/entities.go: tools/getnamedcharrefs.go
	go run $< > $@
