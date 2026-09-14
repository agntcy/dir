// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"encoding/json"
	"errors"
	"fmt"
)

// eventEnvelope is the part of the log's event envelope the resolver reads:
// {"payload":{"producer":{"event":{...}}}}.
type eventEnvelope struct {
	Payload eventPayload `json:"payload"`
}

type eventPayload struct {
	Producer eventProducer `json:"producer"`
}

type eventProducer struct {
	Event *producerEvent `json:"event"`
}

// producerEvent names the agent an event is about. The reference log writes
// ansId; older envelopes wrote agentId.
type producerEvent struct {
	AnsID   string `json:"ansId"`
	AgentID string `json:"agentId"`
	AnsName string `json:"ansName"`
}

func (e *producerEvent) agentID() string {
	if e.AnsID != "" {
		return e.AnsID
	}

	return e.AgentID
}

var errNotAnEvent = errors.New("envelope carries no agent event")

func decodeEvent(raw []byte) (*producerEvent, error) {
	var envelope eventEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode event envelope: %w", err)
	}

	event := envelope.Payload.Producer.Event
	if event == nil || event.agentID() == "" {
		return nil, errNotAnEvent
	}

	return event, nil
}
