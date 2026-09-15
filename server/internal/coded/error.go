package coded

type Error struct {
	HTTP int
	Code string
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func New(httpStatus int, code, msg string) *Error {
	return &Error{HTTP: httpStatus, Code: code, Msg: msg}
}
