package source

type Span struct {
	Start int
	End   int
}

type StringPos struct {
	Text  string
	Start Pos
}

type Pos int
