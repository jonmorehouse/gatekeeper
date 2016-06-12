package shared

import (
	"io"
	"io/ioutil"
	"net/http"
)

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

	// if a plugin would like to overwrite the body being returned, then a
	// reader can be passed along to read data back to the responseWriter
	// instead of using the default. If nil, then we don't copy over.  NOTE
	// if override body is used, its recommended to use the SetBody(reader)
	// method which will accept a reader and write it into a buffer that is
	// compatible over the wire.
	OverrideBody []byte
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
		OverrideBody:     nil,
	}
}

func (r *Response) SetBody(reader io.Reader) error {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	r.OverrideBody = bytes
	r.ContentLength = int64(len(r.OverrideBody))
	return nil
}
