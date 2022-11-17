package parse

import "fmt"

type LineCol struct {
	LineIndex   int // 0-based
	ColumnIndex int // 0-based
}

func (lc *LineCol) String() string {
	return fmt.Sprintf("%d:%d", lc.LineIndex+1, lc.ColumnIndex+1)
}

type Location struct {
	Offset     int
	LineNumber int
	LineOffset int
}

func (l *Location) ColumnNumber() int {
	return 1 + l.LineOffset - l.LineNumber
}
