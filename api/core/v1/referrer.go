// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"
)

func (r *RecordReferrer) GetPublicKey() (string, error) {
	if r.GetData() == nil {
		return "", errors.New("data struct is nil")
	}

	dataMap := r.GetData().AsMap()

	publicKeyValue, ok := dataMap["publicKey"]
	if !ok {
		return "", errors.New("publicKey field not found in data")
	}

	publicKey, ok := publicKeyValue.(string)
	if !ok {
		return "", errors.New("publicKey field is not a string")
	}

	if publicKey == "" {
		return "", errors.New("publicKey field is empty")
	}

	return publicKey, nil
}
