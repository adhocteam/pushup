test tests:
	go test -v ./...

fixme todo:
	@grep -h -E '(TODO|FIXME)' *.go | sed -E -e 's/.*\/\/ (TODO|FIXME)\(paulsmith\): //'
