package dto_test

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestResponsesStreamResponseAcceptsObjectArguments(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "output item",
			payload: `{"type":"response.output_item.done","item":{"type":"function_call","arguments":{"query":"weather","limit":0,"strict":false}}}`,
		},
		{
			name:    "completed response output",
			payload: `{"type":"response.completed","response":{"output":[{"type":"function_call","arguments":{"query":"weather","limit":0,"strict":false}}]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response dto.ResponsesStreamResponse
			require.NoError(t, common.UnmarshalJsonStr(tt.payload, &response))

			encoded, err := common.Marshal(response)
			require.NoError(t, err)
			require.Contains(t, string(encoded), `"arguments":{"query":"weather","limit":0,"strict":false}`)
		})
	}
}
