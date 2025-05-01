package errors

import "errors"

// Standard API-related errors
var (
	ErrUnauthorized       = errors.New("opensubtitles: unauthorized (invalid API key or token)")
	ErrForbidden          = errors.New("opensubtitles: forbidden (insufficient permissions or quota exceeded)")
	ErrNotFound           = errors.New("opensubtitles: resource not found")
	ErrRateLimited        = errors.New("opensubtitles: rate limit exceeded")
	ErrServiceUnavailable = errors.New("opensubtitles: service unavailable or internal server error")

	// Application/Flow specific errors
	ErrNotLoggedIn     = errors.New("client: not logged in")
	ErrUploadDuplicate = errors.New("upload: subtitle is already present in the database (duplicate)")
)

// TODO: Add more specific errors as needed, potentially wrapping underlying errors.
