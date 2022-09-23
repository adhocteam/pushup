all: install

install:
	go install -gcflags="all=-N -l" .

.PHONY: install

build:
	go build -o pushup .

.PHONY: build

build-docker:
	docker build -t pushup .

.PHONY: build-docker

test tests:
	go test -v ./...

.PHONY: test tests

fixme todo:
	@grep -h -E '(TODO|FIXME)' *.go | sed -E -e 's/.*\/\/ (TODO|FIXME)\(paulsmith\): //'

entities.go: tools/getnamedcharrefs.go
	go run $< > $@
