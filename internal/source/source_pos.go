package source

import "strconv"

type Span struct {
	Start int
	Len   int
}

func (s Span) String() string {
	return "Span{Start:" + strconv.Itoa(s.Start) + " Len:" + strconv.Itoa(s.Len) + "}"
}

type StringPos struct {
	Text  string
	Start Pos
}

type Pos int
