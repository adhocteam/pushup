all: pushup

.PHONY: pushup clean install test

pushup:
	go build -gcflags="all=-N -l" -o pushup ./cmd/pushup

clean:
	rm -f pushup

install:
	go install -gcflags="all=-N -l" -v ./cmd/...

test:
	go test ./...

internal/parser/entities.go: tools/getnamedcharrefs.go
	go run $< > $@
