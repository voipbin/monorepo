package requesthandler

import (
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/sock"
	"net/http"
	"reflect"

	"github.com/pkg/errors"
)

var (
	ErrUnknown = errors.New("unknown error")

	// 4xx Client Errors
	ErrBadRequest                  = errors.New(http.StatusText(http.StatusBadRequest))                   // 400 Bad Request
	ErrUnauthorized                = errors.New(http.StatusText(http.StatusUnauthorized))                 // 401 Unauthorized
	ErrPaymentRequired             = errors.New(http.StatusText(http.StatusPaymentRequired))              // 402 Payment Required
	ErrForbidden                   = errors.New(http.StatusText(http.StatusForbidden))                    // 403 Forbidden
	ErrNotFound                    = errors.New(http.StatusText(http.StatusNotFound))                     // 404 Not Found
	ErrMethodNotAllowed            = errors.New(http.StatusText(http.StatusMethodNotAllowed))             // 405 Method Not Allowed
	ErrNotAcceptable               = errors.New(http.StatusText(http.StatusNotAcceptable))                // 406 Not Acceptable
	ErrProxyAuthRequired           = errors.New(http.StatusText(http.StatusProxyAuthRequired))            // 407 Proxy Authentication Required
	ErrRequestTimeout              = errors.New(http.StatusText(http.StatusRequestTimeout))               // 408 Request Timeout
	ErrConflict                    = errors.New(http.StatusText(http.StatusConflict))                     // 409 Conflict
	ErrGone                        = errors.New(http.StatusText(http.StatusGone))                         // 410 Gone
	ErrLengthRequired              = errors.New(http.StatusText(http.StatusLengthRequired))               // 411 Length Required
	ErrPreconditionFailed          = errors.New(http.StatusText(http.StatusPreconditionFailed))           // 412 Precondition Failed
	ErrPayloadTooLarge             = errors.New(http.StatusText(http.StatusRequestEntityTooLarge))        // 413 Payload Too Large (Deprecated, use StatusContentTooLarge)
	ErrURITooLong                  = errors.New(http.StatusText(http.StatusRequestURITooLong))            // 414 URI Too Long
	ErrUnsupportedMediaType        = errors.New(http.StatusText(http.StatusUnsupportedMediaType))         // 415 Unsupported Media Type
	ErrRangeNotSatisfiable         = errors.New(http.StatusText(http.StatusRequestedRangeNotSatisfiable)) // 416 Range Not Satisfiable
	ErrExpectationFailed           = errors.New(http.StatusText(http.StatusExpectationFailed))            // 417 Expectation Failed
	ErrTeapot                      = errors.New(http.StatusText(http.StatusTeapot))                       // 418 I'm a teapot (RFC 2324)
	ErrMisdirectedRequest          = errors.New(http.StatusText(http.StatusMisdirectedRequest))           // 421 Misdirected Request
	ErrUnprocessableEntity         = errors.New(http.StatusText(http.StatusUnprocessableEntity))          // 422 Unprocessable Entity
	ErrLocked                      = errors.New(http.StatusText(http.StatusLocked))                       // 423 Locked
	ErrFailedDependency            = errors.New(http.StatusText(http.StatusFailedDependency))             // 424 Failed Dependency
	ErrUpgradeRequired             = errors.New(http.StatusText(http.StatusUpgradeRequired))              // 426 Upgrade Required
	ErrPreconditionRequired        = errors.New(http.StatusText(http.StatusPreconditionRequired))         // 428 Precondition Required
	ErrTooManyRequests             = errors.New(http.StatusText(http.StatusTooManyRequests))              // 429 Too Many Requests
	ErrRequestHeaderFieldsTooLarge = errors.New(http.StatusText(http.StatusRequestHeaderFieldsTooLarge))  // 431 Request Header Fields Too Large
	ErrUnavailableForLegalReasons  = errors.New(http.StatusText(http.StatusUnavailableForLegalReasons))   // 451 Unavailable For Legal Reasons

	// 5xx Server Errors
	ErrInternal                      = errors.New(http.StatusText(http.StatusInternalServerError))           // 500 Internal Server Error
	ErrNotImplemented                = errors.New(http.StatusText(http.StatusNotImplemented))                // 501 Not Implemented
	ErrBadGateway                    = errors.New(http.StatusText(http.StatusBadGateway))                    // 502 Bad Gateway
	ErrServiceUnavailable            = errors.New(http.StatusText(http.StatusServiceUnavailable))            // 503 Service Unavailable
	ErrGatewayTimeout                = errors.New(http.StatusText(http.StatusGatewayTimeout))                // 504 Gateway Timeout
	ErrHTTPVersionNotSupported       = errors.New(http.StatusText(http.StatusHTTPVersionNotSupported))       // 505 HTTP Version Not Supported
	ErrVariantAlsoNegotiates         = errors.New(http.StatusText(http.StatusVariantAlsoNegotiates))         // 506 Variant Also Negotiates
	ErrInsufficientStorage           = errors.New(http.StatusText(http.StatusInsufficientStorage))           // 507 Insufficient Storage
	ErrLoopDetected                  = errors.New(http.StatusText(http.StatusLoopDetected))                  // 508 Loop Detected
	ErrNotExtended                   = errors.New(http.StatusText(http.StatusNotExtended))                   // 510 Not Extended
	ErrNetworkAuthenticationRequired = errors.New(http.StatusText(http.StatusNetworkAuthenticationRequired)) // 511 Network Authentication Required
)

// HttpStatusErrorMap maps HTTP status codes to their corresponding sentinel error.
// If a status code doesn't have a specific sentinel error, it might not be in this map.
var HttpStatusErrorMap = map[int]error{
	http.StatusBadRequest:                   ErrBadRequest,
	http.StatusUnauthorized:                 ErrUnauthorized,
	http.StatusPaymentRequired:              ErrPaymentRequired,
	http.StatusForbidden:                    ErrForbidden,
	http.StatusNotFound:                     ErrNotFound,
	http.StatusMethodNotAllowed:             ErrMethodNotAllowed,
	http.StatusNotAcceptable:                ErrNotAcceptable,
	http.StatusProxyAuthRequired:            ErrProxyAuthRequired,
	http.StatusRequestTimeout:               ErrRequestTimeout,
	http.StatusConflict:                     ErrConflict,
	http.StatusGone:                         ErrGone,
	http.StatusLengthRequired:               ErrLengthRequired,
	http.StatusPreconditionFailed:           ErrPreconditionFailed,
	http.StatusRequestEntityTooLarge:        ErrPayloadTooLarge, // Deprecated, but here for compatibility
	http.StatusRequestURITooLong:            ErrURITooLong,
	http.StatusUnsupportedMediaType:         ErrUnsupportedMediaType,
	http.StatusRequestedRangeNotSatisfiable: ErrRangeNotSatisfiable,
	http.StatusExpectationFailed:            ErrExpectationFailed,
	http.StatusTeapot:                       ErrTeapot,
	http.StatusMisdirectedRequest:           ErrMisdirectedRequest,
	http.StatusUnprocessableEntity:          ErrUnprocessableEntity,
	http.StatusLocked:                       ErrLocked,
	http.StatusFailedDependency:             ErrFailedDependency,
	http.StatusUpgradeRequired:              ErrUpgradeRequired,
	http.StatusPreconditionRequired:         ErrPreconditionRequired,
	http.StatusTooManyRequests:              ErrTooManyRequests,
	http.StatusRequestHeaderFieldsTooLarge:  ErrRequestHeaderFieldsTooLarge,
	http.StatusUnavailableForLegalReasons:   ErrUnavailableForLegalReasons,

	http.StatusInternalServerError:           ErrInternal,
	http.StatusNotImplemented:                ErrNotImplemented,
	http.StatusBadGateway:                    ErrBadGateway,
	http.StatusServiceUnavailable:            ErrServiceUnavailable,
	http.StatusGatewayTimeout:                ErrGatewayTimeout,
	http.StatusHTTPVersionNotSupported:       ErrHTTPVersionNotSupported,
	http.StatusVariantAlsoNegotiates:         ErrVariantAlsoNegotiates,
	http.StatusInsufficientStorage:           ErrInsufficientStorage,
	http.StatusLoopDetected:                  ErrLoopDetected,
	http.StatusNotExtended:                   ErrNotExtended,
	http.StatusNetworkAuthenticationRequired: ErrNetworkAuthenticationRequired,
}

func getResponseStatusCodeError(statusCode int) error {
	if statusCode < 300 {
		return nil
	}

	err, ok := HttpStatusErrorMap[statusCode]
	if ok {
		return err
	}

	return errors.Wrapf(ErrUnknown, "response code: %d", statusCode)
}

func parseResponse(resp *sock.Response, out any) error {
	if resp == nil {
		// nothing to parse.
		return nil
	}

	if errStatus := getResponseStatusCodeError(resp.StatusCode); errStatus != nil {
		return errStatus
	}

	// if caller doesn't want to parse response data, just return
	if out == nil {
		return nil
	}

	// out must be a pointer
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("out must be a pointer, got %T", out)
	}

	// no data to unmarshal
	if len(resp.Data) == 0 {
		return nil
	}

	if errUnmarshal := json.Unmarshal([]byte(resp.Data), out); errUnmarshal != nil {
		return fmt.Errorf("failed to unmarshal response: %w", errUnmarshal)
	}

	return nil
}
