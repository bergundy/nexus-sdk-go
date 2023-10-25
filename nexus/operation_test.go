package nexus

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartOperation(t *testing.T) {
	handler, err := NewServiceHandler([]UntypedOperationHandler{
		NewSyncOperation("number-validator", func(ctx context.Context, input int, options StartOperationOptions) (int, error) {
			if input == 0 {
				return 0, fmt.Errorf("cannot process 0")
			}
			return input, nil
		}),
	})
	require.NoError(t, err)

	ctx, client, teardown := setup(t, handler)
	defer teardown()

	result, err := client.ExecuteOperation(ctx, "number-validator", 3, ExecuteOperationOptions{})
	require.NoError(t, err)
	var val int
	err = result.Read(&val)
	require.NoError(t, err)
	require.Equal(t, 3, val)
}
