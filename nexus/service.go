package nexus

import (
	"context"
	"fmt"
)

type ServiceHandlerOptions struct {
	Operations []UntypedOperationHandler
	Codec      Codec
}

type ServiceHandler struct {
	UnimplementedHandler

	operations map[string]UntypedOperationHandler
	codec      Codec
}

func NewServiceHandler(options ServiceHandlerOptions) (*ServiceHandler, error) {
	operations := make(map[string]UntypedOperationHandler, len(options.Operations))
	for _, op := range options.Operations {
		if _, found := operations[op.GetName()]; found {
			return nil, fmt.Errorf("duplicate operation: %s", op.GetName())
		}
		operations[op.GetName()] = op
	}
	if options.Codec == nil {
		options.Codec = DefaultCodec{}
	}
	return &ServiceHandler{
		operations: operations,
		codec:      options.Codec,
	}, nil
}

// CancelOperation implements Handler.
func (h *ServiceHandler) CancelOperation(ctx context.Context, request *CancelOperationRequest) error {
	if op, found := h.operations[request.Operation]; found {
		return op.CancelOperation(ctx, request)
	}
	return newNotFoundError("not found")
}

// MapCompletion implements Handler.
func (h *ServiceHandler) MapCompletion(ctx context.Context, request *MapCompletionRequest) (OperationCompletion, error) {
	if op, found := h.operations[request.Operation]; found {
		return op.MapCompletion(ctx, request)
	}
	return nil, newNotFoundError("not found")
}

// GetOperationInfo implements Handler.
func (h *ServiceHandler) GetOperationInfo(ctx context.Context, request *GetOperationInfoRequest) (*OperationInfo, error) {
	if op, found := h.operations[request.Operation]; found {
		return op.GetOperationInfo(ctx, request)
	}
	return nil, newNotFoundError("not found")
}

// GetOperationResult implements Handler.
func (h *ServiceHandler) GetOperationResult(ctx context.Context, request *GetOperationResultRequest) (*OperationResponseSync, error) {
	if op, found := h.operations[request.Operation]; found {
		return op.GetOperationResult(ctx, request)
	}
	return nil, newNotFoundError("not found")
}

// StartOperation implements Handler.
func (h *ServiceHandler) StartOperation(ctx context.Context, request *StartOperationRequest) (OperationResponse, error) {
	if op, found := h.operations[request.Operation]; found {
		return op.StartOperation(ctx, request)
	}
	return nil, newNotFoundError("not found")
}

var _ Handler = &ServiceHandler{}
