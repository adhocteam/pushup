Changelog
=========

v0.2
----

  * Add new "pushup routes" command (#75)
  * Add line number and column to syntax errors in parser (#71)
  * Make Pushup a Nix flake
  * Bugfix: fix child process handling/cleanup (#108)
  * Use Go 1.20's new context.WithCancelCause for reloader (#104)
  * Fix doc link on the Hello world page (Andy Hsieh)
  * Fix typo in scaffold site content
  * Add support for profiling Pushup
  * Add CI linting (Bill Mill)
  * Bugfix: fix interaction between ^for and ^partial in codegen (#92)
  * Mangle variable name for fewer naming collisions (Bill Mill)
  * Remove need for -build-pkg by parsing the go.mod file (#85)
  * Add -out-file option to specify binary output (Bill Mill)
  * Display banner.txt on CLI command (#73) (Germán M.S.O)
  * Bugfix: staticcheck errors (#69) (Bill Mill)
  * Bugfix: missing trailing slashes in CRUD demo (#66) (Fredrik Holmqvist)

v0.1
----

  * Initial public release.
