package shared

import (
	"fmt"
	"log"
)

// shared.Error is an RPC friendly error which is used for transferring errors
// back and forth from plugins and the parent process. Behind the scenes, the
// plugin/* packages are responsible for accepting generic error interfaces,
// casting them to *shared.Error types and then transmitting them over the wire
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

var ProgrammingErrorFatal bool

// ProgrammingError's should not happen in normal operations
func ProgrammingError(msg string) {
	err := fmt.Sprintf("programming error: ", msg)
	if ProgrammingErrorFatal {
		log.Fatal(err)
	} else {
		log.Print(err)
	}
}
