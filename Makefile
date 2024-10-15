all: pushup

pushup:
	go build -o pushup ./cmd/pushup

install:
	go install -v ./cmd/...

test:
	go test ./...
