package parse

import "fmt"

type ErrContent struct {
	Code ErrCode
	What string
}

func (e *ErrContent) Error() string {
	return string(e.Code) + " " + e.What
}

type ErrAtLineCol struct {
	Err error
	Loc LineCol
}

func (e *ErrAtLineCol) Error() string {
	return fmt.Sprintf("[%s] %s", &e.Loc, e.Err.Error())
}

func Expected(v string) *ErrContent     { return &ErrContent{ErrCodeExpected, v} }
func Unexpected(v string) *ErrContent   { return &ErrContent{ErrCodeUnexpected, v} }
func Unterminated(v string) *ErrContent { return &ErrContent{ErrCodeUnterminated, v} }
func Unpaired(v string) *ErrContent     { return &ErrContent{ErrCodeUnpaired, v} }
func Invalid(v string) *ErrContent      { return &ErrContent{ErrCodeInvalid, v} }

type ErrCode string

const (
	ErrCodeNone         = ErrCode("")
	ErrCodeUnexpected   = ErrCode("unexpected")
	ErrCodeExpected     = ErrCode("expected")
	ErrCodeUnterminated = ErrCode("unterminated")
	ErrCodeUnpaired     = ErrCode("unpaired")
	ErrCodeInvalid      = ErrCode("invalid")
)
