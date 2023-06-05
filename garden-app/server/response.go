package server

import (
	"net/http"

	"github.com/go-chi/render"
)

// ErrNotFoundResponse is a basic error response for missing resource
var ErrNotFoundResponse = &ErrResponse{HTTPStatusCode: 404, StatusText: "Resource not found."}

// ErrResponse is a struct used to organize HTTP error responses
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

// Render will verify and render the ErrResponse
func (e *ErrResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

// ErrInvalidRequest creates a 400 ErrResponse for a bad request
func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

// ErrRender creates a 422 response for errors encountered while rendering a response
func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 422,
		StatusText:     "Error rendering response.",
		ErrorText:      err.Error(),
	}
}

// InternalServerError creates a generic 500 error for a server-side error
func InternalServerError(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 500,
		StatusText:     "Server Error.",
		ErrorText:      err.Error(),
	}
}

// Link is used for HATEOAS-style REST hypermedia
type Link struct {
	Rel  string `json:"rel,omitempty"`
	HRef string `json:"href"`
}
