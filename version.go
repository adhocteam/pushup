package main

import (
	"fmt"
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

func printVersion() {
	runtimeVersion := getRuntimeVersion()

	// If the binary was not compiled with a git version, don't print an empty
	// parens
	if runtimeVersion == "" {
		fmt.Printf("Pushup %s\n", VERSION)
	} else {
		fmt.Printf("Pushup %s (%s)\n", VERSION, runtimeVersion[:8])
	}
}
