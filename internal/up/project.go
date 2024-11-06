package up

// Project represents a Pushup project
type Project struct {
	RootDir   string
	Module    Module
	ModuleDir string
	Files     []File
	StaticDir string // path to static assets directory to be embedded
}

type Module struct {
	Path string
}
