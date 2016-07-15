package gatekeeper

import (
	"io"
	"io/ioutil"
	"net/http"
)

type ResponseType uint

const (
	OkResponse ResponseType = iota + 1
	RedirectResponse
	UserErrorResponse
	InternalErrorResponse
)

var responseTypeMapping = map[ResponseType]string{
	OkResponse:            "2xx: ok",
	RedirectResponse:      "3xx: redirect",
	UserErrorResponse:     "4xx: user error",
	InternalErrorResponse: "5xx: internal error",
}

func NewResponseType(statusCode int) ResponseType {
	mapping := map[int]ResponseType{
		2: OkResponse,
		3: RedirectResponse,
		4: UserErrorResponse,
		5: InternalErrorResponse,
	}

	responseType, found := mapping[statusCode/100]
	if !found {
		ProgrammingError("invalid status code for response type")
		return InternalErrorResponse
	}

	return responseType
}

// Response is a rpc compatible representation of an http.Response type which by default, _does_ not pass the _actual_ body of a request over RPC
type Response struct {
	Status     string
	StatusCode int
	Proto      string
	ProtoMajor int
	ProtoMinor int

	Header http.Header

	ContentLength    int64
	TransferEncoding []string
	Close            bool
	Trailer          http.Header

	// if an error has occurred, its attached to the response and used as
	// the body. This is used to add additional context to a response in
	// the case that an error occurred.
	Error *Error

	// if a plugin would like to overwrite the body being returned, then a
	// reader can be passed along to read data back to the responseWriter
	// instead of using the default. If nil, then we don't copy over.  NOTE
	// if override body is used, its recommended to use the SetBody(reader)
	// method which will accept a reader and write it into a buffer that is
	// compatible over the wire.
	Body []byte
}

func NewResponse(resp *http.Response) *Response {
	return &Response{
		Status:           resp.Status,
		StatusCode:       resp.StatusCode,
		Proto:            resp.Proto,
		ProtoMajor:       resp.ProtoMajor,
		ProtoMinor:       resp.ProtoMinor,
		Header:           resp.Header,
		ContentLength:    resp.ContentLength,
		TransferEncoding: resp.TransferEncoding,
		Close:            resp.Close,
		Trailer:          resp.Trailer,
		Body:             nil,
		Error:            nil,
	}
}

func NewErrorResponse(statusCode int, err error) *Response {
	return &Response{
		StatusCode:    statusCode,
		Error:         NewError(err),
		Body:          []byte(err.Error()),
		ContentLength: int64(len(err.Error()) - 1),
	}
}

func (r *Response) SetCode(code int) {
	r.StatusCode = code
	r.Status = http.StatusText(code)
}

func (r *Response) SetBody(reader io.Reader) error {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	r.Body = bytes
	r.ContentLength = int64(len(bytes)) - 1
	return nil
}
