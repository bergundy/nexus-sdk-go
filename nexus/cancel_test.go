package nexus

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type asyncWithCancelHandler struct {
	expectHeader bool
	UnimplementedHandler
}

func (h *asyncWithCancelHandler) StartOperation(ctx context.Context, operation string, input *EncodedStream, options StartOperationOptions) (OperationResponse[any], error) {
	return &OperationResponseAsync{
		OperationID: "a/sync",
	}, nil
}

func (h *asyncWithCancelHandler) CancelOperation(ctx context.Context, operation, operationID string, options CancelOperationOptions) error {
	if operation != "f/o/o" {
		return newBadRequestError("expected operation to be 'foo', got: %s", operation)
	}
	if operationID != "a/sync" {
		return newBadRequestError("expected operation ID to be 'async', got: %s", operationID)
	}
	if h.expectHeader && options.Header.Get("foo") != "bar" {
		return newBadRequestError("invalid 'foo' request header")
	}
	if options.Header.Get("User-Agent") != userAgent {
		return newBadRequestError("invalid 'User-Agent' header: %q", options.Header.Get("User-Agent"))
	}
	return nil
}

func TestCancel_HandleFromStart(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithCancelHandler{expectHeader: true})
	defer teardown()

	result, err := client.StartOperation(ctx, "f/o/o", nil, StartOperationOptions{})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)
	err = handle.Cancel(ctx, CancelOperationOptions{
		Header: http.Header{"foo": []string{"bar"}},
	})
	require.NoError(t, err)
}

func TestCancel_HandleFromClient(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithCancelHandler{})
	defer teardown()

	handle, err := client.NewHandle("f/o/o", "a/sync")
	require.NoError(t, err)
	err = handle.Cancel(ctx, CancelOperationOptions{})
	require.NoError(t, err)
}
