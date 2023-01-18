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
	go test -v . ./_runtime

.PHONY: test tests

fixme todo:
	@grep -h -E '(TODO|FIXME)' *.go | sed -E -e 's/.*\/\/ (TODO|FIXME)\(paulsmith\): //'

entities.go: tools/getnamedcharrefs.go
	go run $< > $@

banner.txt:
	echo '^ Pushup' | figlet -c -k -f lean | tr ' _/' ' //' > $@

lint:
	$(if $(shell command -v golangci-lint 2> /dev/null),$(info),$(error Please install golangci-lint https://golangci-lint.run/usage/install/))
	golangci-lint run

.PHONY: lint

