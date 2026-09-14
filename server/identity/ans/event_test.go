// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeEvent(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantAgentID string
		wantAnsName string
		wantErr     string
	}{
		{
			name:        "ansId names the agent",
			raw:         `{"payload":{"producer":{"event":{"ansId":"` + testAgentID + `","ansName":"` + testAnsName + `"}}}}`,
			wantAgentID: testAgentID,
			wantAnsName: testAnsName,
		},
		{
			name:        "agentId is read when ansId is absent",
			raw:         `{"payload":{"producer":{"event":{"agentId":"` + testAgentID + `","ansName":"` + testAnsName + `"}}}}`,
			wantAgentID: testAgentID,
			wantAnsName: testAnsName,
		},
		{
			name:        "ansId wins over agentId",
			raw:         `{"payload":{"producer":{"event":{"ansId":"` + testAgentID + `","agentId":"` + otherAgentID + `"}}}}`,
			wantAgentID: testAgentID,
		},
		{
			name:    "envelope without an event",
			raw:     `{"payload":{"producer":{}}}`,
			wantErr: "envelope carries no agent event",
		},
		{
			name:    "event without an agent id",
			raw:     `{"payload":{"producer":{"event":{"ansName":"` + testAnsName + `"}}}}`,
			wantErr: "envelope carries no agent event",
		},
		{
			name:    "not json",
			raw:     "not json",
			wantErr: "decode event envelope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := decodeEvent([]byte(tt.raw))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantAgentID, event.agentID())
			assert.Equal(t, tt.wantAnsName, event.AnsName)
		})
	}
}
