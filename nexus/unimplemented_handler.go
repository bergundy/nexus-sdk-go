package nexus

import (
	"context"
)

// UnimplementedHandler must be embedded into any [Handler] implementation for future compatibility.
// It implements all methods on the [Handler] interface, panicking at runtime if they are not implemented by the
// embedding type.
type UnimplementedHandler struct{}

func (h *UnimplementedHandler) mustEmbedUnimplementedHandler() {}

// StartOperation implements the Handler interface.
func (h *UnimplementedHandler) StartOperation(ctx context.Context, operation string, input *EncodedStream, options StartOperationOptions) (OperationResponse, error) {
	return nil, &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// GetOperationResult implements the Handler interface.
func (h *UnimplementedHandler) GetOperationResult(ctx context.Context, operation, operationID string, options GetOperationResultOptions) (any, error) {
	return nil, &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// GetOperationInfo implements the Handler interface.
func (h *UnimplementedHandler) GetOperationInfo(ctx context.Context, operation, operationID string, options GetOperationInfoOptions) (*OperationInfo, error) {
	return nil, &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// CancelOperation implements the Handler interface.
func (h *UnimplementedHandler) CancelOperation(ctx context.Context, operation, operationID string, options CancelOperationOptions) error {
	return &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}
