package main

import (
	"fmt"
	"io"
	"runtime/debug"
)

const (
	VERSION = "0.0.1"
)

// getRuntimeVersion searches the buildinfo built into the binary to find and
// return the git revision, if present. Returns an empty string otherwise.
func getRuntimeVersion() string {
	if bi, ok := debug.ReadBuildInfo(); !ok {
		panic("Unable to get build info")
	} else {
		for i := range bi.Settings {
			if bi.Settings[i].Key == "vcs.revision" {
				return bi.Settings[i].Value
			}
		}
	}
	return ""
}

func printVersion(w io.Writer) {
	runtimeVersion := getRuntimeVersion()

	// If the binary was not compiled with a git version, don't print an empty
	// parens
	if runtimeVersion == "" {
		fmt.Fprintf(w, "Pushup %s\n", VERSION)
	} else {
		fmt.Fprintf(w, "Pushup %s (%s)\n", VERSION, runtimeVersion[:8])
	}
}
