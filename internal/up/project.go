package up

// Project represents a Pushup project
type Project struct {
	RootDir   string
	Module    Module
	ModuleDir string
	Files     []File
}

type Module struct {
	Path string
}
