package utils

type ErrorRepo struct {
	cause string
	msg   string
}

func (e *ErrorRepo) Error() string {
	return e.cause
}

func (e *ErrorRepo) Message() string {
	return e.msg
}

func NewErrorRepo(msg string, err error) *ErrorRepo {
	result := &ErrorRepo{msg: msg}
	if err != nil {
		result.cause = err.Error()
	}
	return result
}
