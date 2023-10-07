package nexus

import (
	"context"
	"errors"
)

type NoResult interface {
	notImplementable()
}

type Operation[I, O any] interface {
	GetName() string
	IO(I, O)
}

type UntypedOperationHandler interface {
	GetName() string
	Handler
}

// OperationHandler is a handler for a single operation.
type OperationHandler[I, O any] interface {
	Operation[I, O]
	UntypedOperationHandler
}

type SyncOperation[I, O any] struct {
	UnimplementedHandler
	service *ServiceHandler

	Name    string
	Handler func(context.Context, I) (O, error)
}

func NewSyncOperation[I any, O any](name string, handler func(context.Context, I) (O, error)) *SyncOperation[I, O] {
	return &SyncOperation[I, O]{
		Name:    name,
		Handler: handler,
	}
}

// GetName implements OperationHandler.
func (h *SyncOperation[I, O]) GetName() string {
	return h.Name
}

func (h *SyncOperation[I, O]) init(s *ServiceHandler) {
	h.service = s
}

// IO implements OperationHandler.
func (*SyncOperation[I, O]) IO(I, O) {}

// StartOperation implements OperationHandler.
func (h *SyncOperation[I, O]) StartOperation(ctx context.Context, request *StartOperationRequest) (OperationResponse, error) {
	dc := GetCodec(ctx)
	var i I
	if err := dc.FromMessage(request.Input, &i); err != nil {
		// log actual error?
		return nil, newBadRequestError("invalid request payload")
	}

	o, err := h.Handler(ctx, i)
	if err != nil {
		return nil, err
	}
	message, err := dc.ToMessage(o)
	return &OperationResponseSync{*message}, err
}

var _ OperationHandler[any, any] = &SyncOperation[any, any]{}

type AsyncOperationHandler[I, O any] interface {
	Start(context.Context, I) (*OperationResponseAsync, error)
	Cancel(context.Context, string) error
	GetResult(context.Context, string) (O, error)
	GetInfo(context.Context, string) (*OperationInfo, error)
}

type AsyncOperation[I, M, O any] struct {
	UnimplementedHandler

	Name         string
	Handler      AsyncOperationHandler[I, M]
	ResultMapper func(context.Context, M, *UnsuccessfulOperationError) (O, *UnsuccessfulOperationError, error)
}

func NewAsyncOperation[I, O any](name string, handler AsyncOperationHandler[I, O]) *AsyncOperation[I, O, O] {
	return &AsyncOperation[I, O, O]{
		Name:    name,
		Handler: handler,
		ResultMapper: func(ctx context.Context, res O, uoe *UnsuccessfulOperationError) (O, *UnsuccessfulOperationError, error) {
			return res, uoe, nil
		},
	}
}

func WithMapper[I, M, O1, O2 any](op *AsyncOperation[I, M, O1], mapper func(context.Context, O1, *UnsuccessfulOperationError) (O2, *UnsuccessfulOperationError, error)) *AsyncOperation[I, M, O2] {
	return &AsyncOperation[I, M, O2]{
		Name:    op.Name,
		Handler: op.Handler,
		ResultMapper: func(ctx context.Context, m M, uoe *UnsuccessfulOperationError) (O2, *UnsuccessfulOperationError, error) {
			o1, uoe, err := op.ResultMapper(ctx, m, uoe)
			if err != nil {
				var o2 O2
				return o2, uoe, err
			}
			return mapper(ctx, o1, uoe)
		},
	}
}

// CancelOperation implements OperationHandler.
func (h *AsyncOperation[I, M, O]) CancelOperation(ctx context.Context, request *CancelOperationRequest) error {
	return h.Handler.Cancel(ctx, request.OperationID)
}

// GetName implements OperationHandler.
func (h *AsyncOperation[I, M, O]) GetName() string {
	return h.Name
}

// GetOperationInfo implements OperationHandler.
func (h *AsyncOperation[I, M, O]) GetOperationInfo(ctx context.Context, request *GetOperationInfoRequest) (*OperationInfo, error) {
	return h.Handler.GetInfo(ctx, request.OperationID)
}

// MapCompletion implements Handler.
func (h *AsyncOperation[I, M, O]) MapCompletion(ctx context.Context, request *MapCompletionRequest) (OperationCompletion, error) {
	dc := GetCodec(ctx)
	var m M
	var uoe *UnsuccessfulOperationError

	switch comp := request.Completion.(type) {
	case *OperationCompletionSuccessful:
		message := Message{Header: comp.Header, Body: comp.Body}
		if err := dc.FromMessage(&message, &m); err != nil {
			return nil, err
		}
	case *OperationCompletionUnsuccessful:
		uoe = &UnsuccessfulOperationError{State: comp.State, Failure: *comp.Failure}
	default:
		return nil, newBadRequestError("bad request")
	}

	o, uoe, err := h.ResultMapper(ctx, m, uoe)
	if err != nil {
		return nil, err
	}
	if uoe != nil {
		return &OperationCompletionUnsuccessful{State: uoe.State, Failure: &uoe.Failure}, nil
	}
	msg, err := dc.ToMessage(o)
	if err != nil {
		return nil, err
	}
	return &OperationCompletionSuccessful{Header: msg.Header, Body: msg.Body}, nil
}

// GetOperationResult implements OperationHandler.
func (h *AsyncOperation[I, M, O]) GetOperationResult(ctx context.Context, request *GetOperationResultRequest) (*OperationResponseSync, error) {
	var uoe *UnsuccessfulOperationError
	// TODO: add wait
	m, err := h.Handler.GetResult(ctx, request.OperationID)
	if err != nil {
		if !errors.As(err, &uoe) {
			return nil, err
		}
	}

	o, uoe, err := h.ResultMapper(ctx, m, uoe)

	if err != nil {
		return nil, err
	}
	msg, err := GetCodec(ctx).ToMessage(o)
	if err != nil {
		return nil, err
	}
	return &OperationResponseSync{*msg}, nil
}

// IO implements OperationHandler.
func (*AsyncOperation[I, M, O]) IO(I, O) {}

// StartOperation implements OperationHandler.
func (h *AsyncOperation[I, M, O]) StartOperation(ctx context.Context, request *StartOperationRequest) (OperationResponse, error) {
	var i I
	if err := GetCodec(ctx).FromMessage(request.Input, &i); err != nil {
		// log actual error?
		return nil, newBadRequestError("invalid request payload")
	}

	// TODO: register for mapping
	return h.Handler.Start(ctx, i)
}

var _ OperationHandler[any, any] = &AsyncOperation[any, any, any]{}
