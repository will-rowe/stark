package config

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CheckResponseErr will check an error that was returned
// by a gRPC server.
func CheckResponseErr(err error) {
	if err == nil {
		return
	}

	// TODO: handle gRPC status code
	if e, ok := status.FromError(err); ok {
		switch e.Code() {
		case codes.PermissionDenied:
			log.Fatalf(e.Message())
		case codes.Internal:
			log.Fatalf("internal error")
		case codes.Aborted:
			log.Fatalf("gRPC aborted the call")
		case codes.NotFound:
			log.Fatalf(e.Message())
		default:
			log.Fatalf(e.Message())
		}
	}

	// unhandled errors
	log.Fatal(err)
}
