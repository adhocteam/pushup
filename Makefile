all: install

install:
	go install .

.PHONY: install

build:
	go build -o pushup .

.PHONY: build

test tests:
	go test -v ./...

.PHONY: test tests

fixme todo:
	@grep -h -E '(TODO|FIXME)' *.go | sed -E -e 's/.*\/\/ (TODO|FIXME)\(paulsmith\): //'
