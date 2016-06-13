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

// Cast errors back to classic errors so we can properly handle nil comparisons
// without having to deal with interface / type comparisons where nil isn't
// predictable
func ErrorToError(e *Error) error {
	if e == nil || e.Message == "" {
		return nil
	}
	return e
}
