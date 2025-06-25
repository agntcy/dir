package grpcuitls

import (
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

func ToApiError(err error) *errdetails.ErrorInfo {
	var extractedErr *ComponentError
	if errors.As(err, &extractedErr) {
		return &errdetails.ErrorInfo{
			Reason:   extractedErr.Component.String(),
			Metadata: extractedErr.Metadata,
		}
	}
}
