// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"errors"
	"strconv"
	"strings"

	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-cid"
	record "github.com/libp2p/go-libp2p-record"
)

var validatorLogger = logging.Logger("routing/validators")

// BaseValidator provides common validation logic for all label validators.
type BaseValidator struct{}

// validateKeyFormat validates the basic DHT key format.
func (v *BaseValidator) validateKeyFormat(key string, expectedNamespace string) ([]string, error) {
	// Parse key format: /<namespace>/<specific_path>/<cid>
	// Minimum parts: ["", "namespace", "path", "cid"]
	parts := strings.Split(key, "/")
	if len(parts) < MinLabelKeyParts {
		return nil, errors.New("invalid key format: expected /<namespace>/<specific_path>/<cid>")
	}

	// Validate namespace
	if parts[1] != expectedNamespace {
		return nil, errors.New("invalid namespace: expected " + expectedNamespace + ", got " + parts[1])
	}

	// Extract and validate CID (last part)
	cidStr := parts[len(parts)-1]
	if cidStr == "" {
		return nil, errors.New("missing CID in key")
	}

	// Validate CID format
	_, err := cid.Decode(cidStr)
	if err != nil {
		return nil, errors.New("invalid CID format: " + err.Error())
	}

	return parts, nil
}

// validateValue validates the DHT value (if present).
func (v *BaseValidator) validateValue(value []byte) error {
	if len(value) > 0 {
		// Value should be a valid CID if present
		_, err := cid.Decode(string(value))
		if err != nil {
			return errors.New("invalid CID in value: " + err.Error())
		}
	}

	return nil
}

// selectFirstValid provides default selection logic for all validators.
func (v *BaseValidator) selectFirstValid(key string, values [][]byte, validateFunc func(string, []byte) error) (int, error) {
	validatorLogger.Debug("Selecting from multiple DHT record values", "key", key, "count", len(values))

	if len(values) == 0 {
		return -1, errors.New("no values to select from")
	}

	for i, value := range values {
		err := validateFunc(key, value)
		if err == nil {
			validatorLogger.Debug("Selected DHT record value", "key", key, "index", i)

			return i, nil
		}
	}

	validatorLogger.Warn("No valid values found for DHT record", "key", key)

	return -1, errors.New("no valid values found")
}

// SkillValidator validates DHT records for skill-based content discovery.
type SkillValidator struct {
	BaseValidator
}

// Validate validates a skills DHT record.
// Key format: /skills/<category>/<class>/<cid>
// Future: Can validate against skill taxonomy, required levels, etc.
func (v *SkillValidator) Validate(key string, value []byte) error {
	validatorLogger.Debug("Validating skills DHT record", "key", key)

	// Basic format validation
	parts, err := v.validateKeyFormat(key, NamespaceSkills.String())
	if err != nil {
		return err
	}

	// Skills-specific validation
	if err := v.validateSkillsSpecific(parts); err != nil {
		return err
	}

	// Value validation
	if err := v.validateValue(value); err != nil {
		return err
	}

	validatorLogger.Debug("Skills DHT record validation successful", "key", key)

	return nil
}

// validateSkillsSpecific performs skills-specific validation logic.
func (v *SkillValidator) validateSkillsSpecific(parts []string) error {
	// parts[0] = "", parts[1] = "skills", parts[2:len-1] = skill path components, parts[len-1] = cid
	// Minimum: /skills/<skill_path>/<cid> = MinLabelKeyParts
	// Examples: /skills/golang/CID or /skills/programming/golang/CID
	if len(parts) < MinLabelKeyParts {
		return errors.New("skills key must have format: /skills/<skill_path>/<cid>")
	}

	// Extract skill path (everything between "skills" and CID)
	skillParts := parts[2 : len(parts)-1]
	if len(skillParts) == 0 {
		return errors.New("skill path cannot be empty")
	}

	// Validate that none of the skill path components are empty
	for i, part := range skillParts {
		if part == "" {
			return errors.New("skill path component cannot be empty at position " + strconv.Itoa(i+1))
		}
	}

	// Future: validate against skill taxonomy
	// skillPath := strings.Join(skillParts, "/")
	// if !v.isValidSkillPath(skillPath) {
	//     return errors.New("invalid skill path: " + skillPath)
	// }

	return nil
}

// Select chooses between multiple values for skills records.
func (v *SkillValidator) Select(key string, values [][]byte) (int, error) {
	return v.selectFirstValid(key, values, v.Validate)
}

// DomainValidator validates DHT records for domain-based content discovery.
type DomainValidator struct {
	BaseValidator
}

// Validate validates a domains DHT record.
// Key format: /domains/<domain_path>/<cid>
// Future: Can validate against domain ontology, registry, etc.
func (v *DomainValidator) Validate(key string, value []byte) error {
	validatorLogger.Debug("Validating domains DHT record", "key", key)

	// Basic format validation
	parts, err := v.validateKeyFormat(key, NamespaceDomains.String())
	if err != nil {
		return err
	}

	// Domains-specific validation
	if err := v.validateDomainsSpecific(parts); err != nil {
		return err
	}

	// Value validation
	if err := v.validateValue(value); err != nil {
		return err
	}

	validatorLogger.Debug("Domains DHT record validation successful", "key", key)

	return nil
}

// validateDomainsSpecific performs domains-specific validation logic.
func (v *DomainValidator) validateDomainsSpecific(parts []string) error {
	// parts[0] = "", parts[1] = "domains", parts[2+] = domain path components, last = cid
	if len(parts) < MinLabelKeyParts {
		return errors.New("domains key must have format: /domains/<domain_path>/<cid>")
	}

	// Extract domain path (everything between "domains" and CID)
	domainParts := parts[2 : len(parts)-1]
	if len(domainParts) == 0 {
		return errors.New("domain path cannot be empty")
	}

	// Future: validate against domain registry/ontology
	// domain := strings.Join(domainParts, "/")
	// if !v.isValidDomain(domain) {
	//     return errors.New("invalid domain: " + domain)
	// }

	return nil
}

// Select chooses between multiple values for domains records.
func (v *DomainValidator) Select(key string, values [][]byte) (int, error) {
	return v.selectFirstValid(key, values, v.Validate)
}

// FeatureValidator validates DHT records for feature-based content discovery.
type FeatureValidator struct {
	BaseValidator
}

// Validate validates a features DHT record.
// Key format: /features/<feature_path>/<cid>
// Future: Can validate against feature specifications, versions, etc.
func (v *FeatureValidator) Validate(key string, value []byte) error {
	validatorLogger.Debug("Validating features DHT record", "key", key)

	// Basic format validation
	parts, err := v.validateKeyFormat(key, NamespaceFeatures.String())
	if err != nil {
		return err
	}

	// Features-specific validation
	if err := v.validateFeaturesSpecific(parts); err != nil {
		return err
	}

	// Value validation
	if err := v.validateValue(value); err != nil {
		return err
	}

	validatorLogger.Debug("Features DHT record validation successful", "key", key)

	return nil
}

// validateFeaturesSpecific performs features-specific validation logic.
func (v *FeatureValidator) validateFeaturesSpecific(parts []string) error {
	// parts[0] = "", parts[1] = "features", parts[2+] = feature path components, last = cid
	if len(parts) < MinLabelKeyParts {
		return errors.New("features key must have format: /features/<feature_path>/<cid>")
	}

	// Extract feature path (everything between "features" and CID)
	featureParts := parts[2 : len(parts)-1]
	if len(featureParts) == 0 {
		return errors.New("feature path cannot be empty")
	}

	// Future: validate against feature specifications
	// feature := strings.Join(featureParts, "/")
	// if !v.isValidFeature(feature) {
	//     return errors.New("invalid feature: " + feature)
	// }

	return nil
}

// Select chooses between multiple values for features records.
func (v *FeatureValidator) Select(key string, values [][]byte) (int, error) {
	return v.selectFirstValid(key, values, v.Validate)
}

// CreateLabelValidators creates separate validators for each label namespace.
func CreateLabelValidators() map[string]record.Validator {
	return map[string]record.Validator{
		NamespaceSkills.String():   &SkillValidator{},
		NamespaceDomains.String():  &DomainValidator{},
		NamespaceFeatures.String(): &FeatureValidator{},
	}
}

// ValidateLabelKey validates a label key format before storing in DHT.
func ValidateLabelKey(key string) error {
	parts := strings.Split(key, "/")
	if len(parts) < MinLabelKeyParts {
		return errors.New("invalid key format: expected /<namespace>/<label_path>/<cid>")
	}

	namespace := parts[1]
	if _, valid := ParseNamespace(namespace); !valid {
		return errors.New("unsupported namespace: " + namespace)
	}

	// Extract and validate CID (last part)
	cidStr := parts[len(parts)-1]
	if cidStr == "" {
		return errors.New("missing CID in key")
	}

	_, err := cid.Decode(cidStr)
	if err != nil {
		return errors.New("invalid CID format: " + err.Error())
	}

	return nil
}

// FormatLabelKey formats a label and CID into a proper DHT key.
func FormatLabelKey(label, cidStr string) string {
	// Ensure label starts with /
	if !strings.HasPrefix(label, "/") {
		label = "/" + label
	}

	// Ensure no double slashes and add CID
	key := strings.TrimSuffix(label, "/") + "/" + cidStr

	return key
}

// Example: "/skills/golang/CID123" → "CID123", nil.
func ExtractCIDFromLabelKey(labelKey string) (string, error) {
	parts := strings.Split(labelKey, "/")
	if len(parts) < MinLabelKeyParts {
		return "", errors.New("invalid label key format: expected /<namespace>/<label_path>/<cid>")
	}

	// Validate it's a proper namespace key
	if !IsValidNamespaceKey(labelKey) {
		return "", errors.New("invalid namespace in label key")
	}

	// Extract and validate CID (last part)
	cidStr := parts[len(parts)-1]
	if cidStr == "" {
		return "", errors.New("missing CID in label key")
	}

	// Validate CID format
	_, err := cid.Decode(cidStr)
	if err != nil {
		return "", errors.New("invalid CID format: " + err.Error())
	}

	return cidStr, nil
}
