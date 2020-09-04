package status

import (
	"net/http"
	"runtime"
	"strings"

	epb "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
)

func NotFound(err error) bool {
	se, ok := FromError(err)
	if !ok {
		return false
	}

	return se.Code() == codes.NotFound
}

func Errorl(c codes.Code, msg string) error {
	if c == codes.OK {
		return nil
	}

	if !debug {
		return Error(c, msg)
	}

	s := New(c, msg)
	s, _ = s.WithDetails(&epb.DebugInfo{Detail: string(stacks(false))})
	return s.Err()
}

func GetError(err error) string {
	if err == nil {
		return ""
	}

	se, ok := FromError(err)
	if !ok {
		return err.Error()
	}

	return se.Errorl()
}

func GetDetail(err error) string {
	if err == nil {
		return ""
	}

	se, ok := FromError(err)
	if !ok {
		return err.Error()
	}

	var s []string
	details := se.Details()
	for i := range details {
		if info, ok := details[i].(*epb.DebugInfo); ok {
			s = append(s, info.Detail)
		}
	}

	return strings.Join(s, "\n")
}

// stacks is a wrapper for runtime.Stack that attempts to recover the data for all goroutines.
func stacks(all bool) []byte {
	// We don't know how big the traces are, so grow a few times if they don't fit. Start large, though.
	n := 10000
	if all {
		n = 100000
	}
	var trace []byte
	for i := 0; i < 5; i++ {
		trace = make([]byte, n)
		nbytes := runtime.Stack(trace, all)
		if nbytes < len(trace) {
			return trace[:nbytes]
		}
		n *= 2
	}
	return trace
}

func HTTPStatusFromCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	}

	return http.StatusInternalServerError
}
