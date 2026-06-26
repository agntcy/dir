// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package exportfmt

import (
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/translator"
)

func init() {
	RegisterFormatter(FormatAgentSkillBundle, &skillBundleFormatter{})
}

type skillBundleFormatter struct{}

func (f *skillBundleFormatter) Format(record *corev1.Record) ([]byte, error) {
	data := record.GetData()
	if data == nil {
		return nil, fmt.Errorf("record contains no data")
	}

	archive, err := translator.RecordToSkillBundle(data)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to translate record to skill bundle: %w", ErrUnsupportedRecord, err)
	}

	return archive, nil
}

func (f *skillBundleFormatter) FileExtension() string {
	return ExtGzip
}
