package webdav

import (
	"fmt"
	"net/http"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/status"
)

type Error struct {
	HttpStatusCode int
	Context        string
	Err            error
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Context, e.Err)
}

func ErrorFromGrpc(e error, context string) Error {
	s, ok := status.FromError(e)
	if !ok {
		return Error{http.StatusInternalServerError, context, e}
	}
	return Error{gwruntime.HTTPStatusFromCode(s.Code()), context, e}
}

func WriteError(w http.ResponseWriter, err error) {
	if mye, ok := err.(Error); ok {
		http.Error(w, mye.Error(), mye.HttpStatusCode)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
