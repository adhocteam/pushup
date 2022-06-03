test:
	go test -v ./...

fixme todo:
	grep -E '(TODO|FIXME)' *.go
