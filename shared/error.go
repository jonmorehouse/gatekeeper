package shared

type Error struct {
	Message string
}

func NewError(err error) *Error {
	if err == nil {
		return nil
	}

	return &Error{Message: err.Error()}
}

func (e *Error) Error() string {
	return e.Message
}
