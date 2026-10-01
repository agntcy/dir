// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

const (
	// AnnotationKeyIdentity is the record annotation holding its own claimed identity URI.
	AnnotationKeyIdentity = "agntcy.dir/identity"

	// AnnotationKeyOwner is the record annotation holding its claimed owning entity's identity URI.
	AnnotationKeyOwner = "agntcy.dir/owner"
)

// GetIdentity extracts the record's own claimed identity URI from its annotations.
func (r *Record) GetIdentity() string {
	return r.GetAnnotation(AnnotationKeyIdentity)
}

// GetOwner extracts the record's claimed owner identity URI from its annotations.
func (r *Record) GetOwner() string {
	return r.GetAnnotation(AnnotationKeyOwner)
}
