# testdata

This directory contains samples of Pushup pages that test various aspects or
features of the framework. They are intended to be somewhat of end-to-end
tests, in that each Pushup page is separately compiled and run in a test
server, requested by the test client, and its output is compared to its
matching `*.out` file, byte for byte. See `TestPushup` in `main_test.go`
in the main directory for details.
