package nexus

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type asyncWithInfoHandler struct {
	UnimplementedHandler
	expectHeader bool
}

func (h *asyncWithInfoHandler) StartOperation(ctx context.Context, operation string, input *EncodedStream, options StartOperationOptions) (OperationResponse, error) {
	return &OperationResponseAsync{
		OperationID: "needs /URL/ escaping",
	}, nil
}

func (h *asyncWithInfoHandler) GetOperationInfo(ctx context.Context, operation, operationID string, options GetOperationInfoOptions) (*OperationInfo, error) {
	if operation != "escape/me" {
		return nil, newBadRequestError("expected operation to be 'escape me', got: %s", operation)
	}
	if operationID != "needs /URL/ escaping" {
		return nil, newBadRequestError("expected operation ID to be 'needs URL escaping', got: %s", operationID)
	}
	if h.expectHeader && options.Header.Get("foo") != "bar" {
		return nil, newBadRequestError("invalid 'foo' request header")
	}
	if options.Header.Get("User-Agent") != userAgent {
		return nil, newBadRequestError("invalid 'User-Agent' header: %q", options.Header.Get("User-Agent"))
	}
	return &OperationInfo{
		ID:    operationID,
		State: OperationStateCanceled,
	}, nil
}

func TestGetHandlerFromStartInfoHeader(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoHandler{expectHeader: true})
	defer teardown()

	result, err := client.StartOperation(ctx, "escape/me", nil, StartOperationOptions{})
	require.NoError(t, err)
	handle := result.Pending
	require.NotNil(t, handle)
	info, err := handle.GetInfo(ctx, GetOperationInfoOptions{
		Header: http.Header{"foo": []string{"bar"}},
	})
	require.NoError(t, err)
	require.Equal(t, handle.ID, info.ID)
	require.Equal(t, OperationStateCanceled, info.State)
}

func TestGetInfoHandleFromClientNoHeader(t *testing.T) {
	ctx, client, teardown := setup(t, &asyncWithInfoHandler{})
	defer teardown()

	handle, err := client.NewHandle("escape/me", "needs /URL/ escaping")
	require.NoError(t, err)
	info, err := handle.GetInfo(ctx, GetOperationInfoOptions{})
	require.NoError(t, err)
	require.Equal(t, handle.ID, info.ID)
	require.Equal(t, OperationStateCanceled, info.State)
}
