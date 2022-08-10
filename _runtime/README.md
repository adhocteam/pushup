# runtime files for Pushup apps

**NOTE: the need for this directory is temporary and will go away soon.**

This directory contains a few .go files that are necessary for compiled
Pushup apps to run, like the logic that ties routes and pages together,
and the app's main() function. They are copied in to Pushup projects for the
build step. A better approach is to have this functionality be provided as a
package, and then just a thin shim of a main.go can be generated. Then this
directory can go away.

It is so named `_runtime` with a leading underscore so that it will be
skipped by the Go tool in the parent directory.
