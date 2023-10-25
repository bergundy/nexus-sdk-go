package nexus

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type Void interface{}

type Operation[I, O any] interface {
	GetName() string
	IO(I, O)
}

type UntypedOperationHandler interface {
	GetName() string
	mustEmbedUnimplementedOperationHandler()
}

// OperationHandler is a handler for a single operation.
type OperationHandler[I, M, O any] interface {
	Operation[I, O]
	Start(context.Context, I, StartOperationOptions) (OperationResponse[M], error)
	Cancel(context.Context, string, CancelOperationOptions) error
	GetResult(context.Context, string, GetOperationResultOptions) (O, error)
	GetInfo(context.Context, string, GetOperationInfoOptions) (*OperationInfo, error)
	// TODO: this is not implemented yet
	MapResult(context.Context, M, error) (O, error)
}

type UnimplementedOperationHandler[I, M, O any] struct{}

// Cancel implements OperationHandler.
func (*UnimplementedOperationHandler[I, M, O]) Cancel(context.Context, string, CancelOperationOptions) error {
	panic("unimplemented")
}

// GetInfo implements OperationHandler.
func (*UnimplementedOperationHandler[I, M, O]) GetInfo(context.Context, string, GetOperationInfoOptions) (*OperationInfo, error) {
	panic("unimplemented")
}

// GetName implements OperationHandler.
func (*UnimplementedOperationHandler[I, M, O]) GetName() string {
	panic("unimplemented")
}

// GetResult implements OperationHandler.
func (*UnimplementedOperationHandler[I, M, O]) GetResult(context.Context, string, GetOperationResultOptions) (O, error) {
	panic("unimplemented")
}

// IO implements OperationHandler.
func (*UnimplementedOperationHandler[I, M, O]) IO(I, O) {
	panic("unimplemented")
}

// MapResult implements OperationHandler.
func (*UnimplementedOperationHandler[I, M, O]) MapResult(context.Context, M, error) (O, error) {
	panic("unimplemented")
}

// Start implements OperationHandler.
func (*UnimplementedOperationHandler[I, M, O]) Start(context.Context, I, StartOperationOptions) (OperationResponse[M], error) {
	panic("unimplemented")
}

func (*UnimplementedOperationHandler[I, M, O]) mustEmbedUnimplementedOperationHandler() {}

var _ OperationHandler[any, any, any] = &UnimplementedOperationHandler[any, any, any]{}

type SyncOperation[I, O any] struct {
	UnimplementedOperationHandler[I, O, O]

	Name    string
	Handler func(context.Context, I, StartOperationOptions) (O, error)
}

func NewSyncOperation[I any, O any](name string, handler func(context.Context, I, StartOperationOptions) (O, error)) *SyncOperation[I, O] {
	return &SyncOperation[I, O]{
		Name:    name,
		Handler: handler,
	}
}

// GetName implements OperationHandler.
func (h *SyncOperation[I, O]) GetName() string {
	return h.Name
}

// IO implements OperationHandler.
func (*SyncOperation[I, O]) IO(I, O) {}

// StartOperation implements OperationHandler.
func (h *SyncOperation[I, O]) Start(ctx context.Context, input I, options StartOperationOptions) (OperationResponse[O], error) {
	o, err := h.Handler(ctx, input, options)
	if err != nil {
		return nil, err
	}
	return &OperationResponseSync[O]{o}, err
}

var _ OperationHandler[any, any, any] = &SyncOperation[any, any]{}

type ServiceHandler struct {
	UnimplementedHandler

	operations map[string]UntypedOperationHandler
}

func NewServiceHandler(operations []UntypedOperationHandler) (*ServiceHandler, error) {
	mapped := make(map[string]UntypedOperationHandler, len(operations))
	if len(operations) == 0 {
		return nil, errors.New("must provide at least one operation")
	}
	dups := []string{}

	for _, op := range operations {
		if _, found := mapped[op.GetName()]; found {
			dups = append(dups, op.GetName())
		}
		mapped[op.GetName()] = op
	}
	if len(dups) > 0 {
		return nil, fmt.Errorf("duplicate operations: %s", strings.Join(dups, ", "))
	}
	return &ServiceHandler{operations: mapped}, nil
}

// CancelOperation implements Handler.
func (*ServiceHandler) CancelOperation(ctx context.Context, operation string, operationID string, options CancelOperationOptions) error {
	panic("unimplemented")
}

// GetOperationInfo implements Handler.
func (*ServiceHandler) GetOperationInfo(ctx context.Context, operation string, operationID string, options GetOperationInfoOptions) (*OperationInfo, error) {
	panic("unimplemented")
}

// GetOperationResult implements Handler.
func (*ServiceHandler) GetOperationResult(ctx context.Context, operation string, operationID string, options GetOperationResultOptions) (any, error) {
	panic("unimplemented")
}

// StartOperation implements Handler.
func (s *ServiceHandler) StartOperation(ctx context.Context, operation string, input *EncodedStream, options StartOperationOptions) (OperationResponse[any], error) {
	if h, ok := s.operations[operation]; ok {
		m, _ := reflect.TypeOf(h).MethodByName("Start")
		inputType := m.Type.In(2)
		iptr := reflect.New(inputType).Interface()
		if err := input.Read(iptr); err != nil {
			return nil, err
		}
		i := reflect.ValueOf(iptr).Elem()

		values := m.Func.Call([]reflect.Value{reflect.ValueOf(h), reflect.ValueOf(ctx), i, reflect.ValueOf(options)})
		if !values[1].IsNil() {
			return nil, values[1].Interface().(error)
		}
		ret := values[0].Interface()
		return ret.(OperationResponse[any]), nil
	}
	return nil, fmt.Errorf("wow")
}

var _ Handler = &ServiceHandler{}
