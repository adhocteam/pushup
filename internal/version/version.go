package version

import (
	"runtime/debug"
)

func Version() string {
	return runtimeVersion()
}

// runtimeVersion searches the buildinfo built into the binary to find and
// return the git revision, if present. Returns an empty string otherwise.
func runtimeVersion() string {
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
