package parse

import "fmt"

type ErrContent struct {
	Code ErrCode
	What string
}

func (e *ErrContent) Error() string {
	return e.Code.String() + " " + e.What
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

type ErrCode int

const (
	ErrCodeUnmatched = ErrCode(-1)
	ErrCodeNone      = ErrCode(iota)
	ErrCodeUnexpected
	ErrCodeExpected
	ErrCodeUnterminated
	ErrCodeIncomplete
	ErrCodeUnpaired
	ErrCodeInvalid
	ErrCodeOverflow
)

func (ec ErrCode) String() string {
	switch ec {
	case ErrCodeNone:
		return ""
	case ErrCodeUnexpected:
		return "unexpected"
	case ErrCodeExpected:
		return "expected"
	case ErrCodeUnterminated:
		return "unterminated"
	case ErrCodeIncomplete:
		return "incomplete"
	case ErrCodeUnpaired:
		return "unpaired"
	case ErrCodeInvalid:
		return "invalid"
	case ErrCodeOverflow:
		return "overflow"
	default:
		return "<unknown>"
	}
}
