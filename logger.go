package middleware

import (
	"bytes"
	"fmt"
	"github.com/72sevenzy2/http-router/core"
	"io"
	"net/http"
	"time"
)

// custom responseWriter type to capture status code and request byte size.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

// override WriteHeader with custom response writer struct
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code // saving status code in struct
	rw.ResponseWriter.WriteHeader(code)
}

// overriding Write to capture byte size
func (rw *responseWriter) Write(b []byte) (int, error) {
	v, err := rw.ResponseWriter.Write(b)
	rw.size += v // tracking the size in bytes
	return v, err
}

// custom limited writer function for Logger() to limit body size reading.
type LimitedBuffer struct {
	buf   *bytes.Buffer
	limit uint32
}

// custom write func for LimitedBuffer, (allocates new slice based on truncated size on original slice)
func (l *LimitedBuffer) Write(p []byte) (int, error) {
	remaining := l.limit - uint32(l.buf.Len())

	if remaining <= 0 {
		return len(p), nil
	}

	if len(p) > int(remaining) { // check if []byte that is being written to l.buf exceeds remaining.
		p = p[:remaining] // truncate
	}

	return l.buf.Write(p)
}

// plain structs to work with default values.
type bodySize struct {
	size uint32
}

func Logger(confSize uint32) core.Middleware { // returns the middleware type
	return func(hf core.HandlerFunc) core.HandlerFunc {
		return func(w http.ResponseWriter, r *core.Request) {
			start := time.Now() // setting the current time (before the request has ended)
			fmt.Printf("Request has started at: %s, with request method: %s, path: %s\n", start, r.Method, r.URL)

			// buffer for comparison in limited writer Write().
			buf := bytes.Buffer{}

			// setting default value first for request body size

			var opt bodySize
			// only set if confSize was set to 0 (will indicate to user in docs):
			opt = bodySize{
				size: 1024, // default
			}

			if confSize != 0 {
				opt = bodySize{
					size: confSize, // custom size
				}
			}

			// limit size
			lm := &LimitedBuffer{
				buf:   &buf,
				limit: opt.size + 1,
			}

			r.Body = io.NopCloser(io.TeeReader(r.Body, lm)) // using io.NopCloser as io.TeeReader does not implement io.ReadCloser.
			// io.TeeReader allows the current handler to read the request body data, whilst also allowing copying.

			rw := &responseWriter{ // default status code and custom response writer initialisation
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			hf(rw, r)
			// by calling hf() before printing, we give time to the io Readers above to read the request body data.

			// truncating if over 1 kb
			body := buf.Bytes()
			if uint32(len(body)) > opt.size {
				body = body[:opt.size] // truncated
				fmt.Println("body truncated.")
			}

			endTime := time.Since(start) // after the request has ended, in which we will print below
			fmt.Printf("Request ended at: %s ||| Status code: %d ||| Response body size (bytes): %d", endTime, rw.status, rw.size)

			fmt.Println("\n Request body size (bytes): ", opt.size)
			fmt.Println(string(body))

			// redacting sensitive header before printing
			header := r.Header.Clone()
			header.Del("Authorization")

			fmt.Println("Request headers:")
			for k, v := range header {
				fmt.Println(k, v)
			}
		}
	}
}
