// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"errors"
	"strconv"
	"strings"

	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-cid"
	record "github.com/libp2p/go-libp2p-record"
)

// Import routing utilities for label validation
// Note: Since validators is a sub-package of routing, it can import from the parent

// IsValidLabelKey checks if a key starts with any valid label type prefix.
func IsValidLabelKey(key string) bool {
	for _, labelType := range types.AllLabelTypes() {
		if strings.HasPrefix(key, labelType.Prefix()) {
			return true
		}
	}

	return false
}

var validatorLogger = logging.Logger("routing/validators")

// BaseValidator provides common validation logic for all label validators.
type BaseValidator struct{}

// validateKeyFormat validates the enhanced DHT key format with PeerID.
func (v *BaseValidator) validateKeyFormat(key string, expectedNamespace string) ([]string, error) {
	// Parse enhanced key format: /<namespace>/<specific_path>/<cid>/<peer_id>
	// Minimum parts: ["", "namespace", "path", "cid", "peer_id"]
	parts := strings.Split(key, "/")
	if len(parts) < types.MinLabelKeyParts {
		return nil, errors.New("invalid key format: expected /<namespace>/<specific_path>/<cid>/<peer_id>")
	}

	// Validate namespace
	if parts[1] != expectedNamespace {
		return nil, errors.New("invalid namespace: expected " + expectedNamespace + ", got " + parts[1])
	}

	// Extract and validate PeerID (last part) first
	peerID := parts[len(parts)-1]
	if peerID == "" {
		return nil, errors.New("missing PeerID in key")
	}

	// Check if the last part looks like a CID (common mistake)
	if _, err := cid.Decode(peerID); err == nil {
		return nil, errors.New("invalid key format: expected /<namespace>/<specific_path>/<cid>/<peer_id>")
	}

	// Extract and validate CID (second to last part)
	cidStr := parts[len(parts)-2]
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
	parts, err := v.validateKeyFormat(key, types.LabelTypeSkill.String())
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
	// parts[0] = "", parts[1] = "skills", parts[2:len-2] = skill path components, parts[len-2] = cid, parts[len-1] = peer_id
	// Enhanced format: /skills/<skill_path>/<cid>/<peer_id>
	if len(parts) < types.MinLabelKeyParts {
		return errors.New("skills key must have format: /skills/<skill_path>/<cid>/<peer_id>")
	}

	// Extract skill path (everything between "skills" and CID)
	skillParts := parts[2 : len(parts)-2] // Exclude CID and PeerID
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
	parts, err := v.validateKeyFormat(key, types.LabelTypeDomain.String())
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
	// parts[0] = "", parts[1] = "domains", parts[2:len-2] = domain path components, parts[len-2] = cid, parts[len-1] = peer_id
	// Enhanced format: /domains/<domain_path>/<cid>/<peer_id>
	if len(parts) < types.MinLabelKeyParts {
		return errors.New("domains key must have format: /domains/<domain_path>/<cid>/<peer_id>")
	}

	// Extract domain path (everything between "domains" and CID)
	domainParts := parts[2 : len(parts)-2] // Exclude CID and PeerID
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

// ModuleValidator validates DHT records for module-based content discovery.
type ModuleValidator struct {
	BaseValidator
}

// Validate validates a modules DHT record.
// Key format: /modules/<module_path>/<cid>
// Future: Can validate against module specifications, versions, etc.
func (v *ModuleValidator) Validate(key string, value []byte) error {
	validatorLogger.Debug("Validating modules DHT record", "key", key)

	// Basic format validation
	parts, err := v.validateKeyFormat(key, types.LabelTypeModule.String())
	if err != nil {
		return err
	}

	// Modules-specific validation
	if err := v.validateModulesSpecific(parts); err != nil {
		return err
	}

	// Value validation
	if err := v.validateValue(value); err != nil {
		return err
	}

	validatorLogger.Debug("Modules DHT record validation successful", "key", key)

	return nil
}

// validateModulesSpecific performs modules-specific validation logic.
func (v *ModuleValidator) validateModulesSpecific(parts []string) error {
	// parts[0] = "", parts[1] = "modules", parts[2:len-2] = module path components, parts[len-2] = cid, parts[len-1] = peer_id
	// Enhanced format: /modules/<module_path>/<cid>/<peer_id>
	if len(parts) < types.MinLabelKeyParts {
		return errors.New("modules key must have format: /modules/<module_path>/<cid>/<peer_id>")
	}

	// Extract module path (everything between "modules" and CID)
	moduleParts := parts[2 : len(parts)-2] // Exclude CID and PeerID
	if len(moduleParts) == 0 {
		return errors.New("module path cannot be empty")
	}

	// Future: validate against module specifications
	// module := strings.Join(moduleParts, "/")
	// if !v.isValidModule(module) {
	//     return errors.New("invalid module: " + module)
	// }

	return nil
}

// Select chooses between multiple values for modules records.
func (v *ModuleValidator) Select(key string, values [][]byte) (int, error) {
	return v.selectFirstValid(key, values, v.Validate)
}

// LocatorValidator validates DHT records for locator-based content discovery.
type LocatorValidator struct {
	BaseValidator
}

// Validate validates a locators DHT record.
// Key format: /locators/<locator_type>/<cid>/<peer_id>
// Future: Can validate against supported locator types, registry, etc.
func (v *LocatorValidator) Validate(key string, value []byte) error {
	validatorLogger.Debug("Validating locators DHT record", "key", key)

	// Basic format validation
	parts, err := v.validateKeyFormat(key, types.LabelTypeLocator.String())
	if err != nil {
		return err
	}

	// Locators-specific validation
	if err := v.validateLocatorsSpecific(parts); err != nil {
		return err
	}

	// Value validation
	if err := v.validateValue(value); err != nil {
		return err
	}

	validatorLogger.Debug("Locators DHT record validation successful", "key", key)

	return nil
}

// validateLocatorsSpecific performs locators-specific validation logic.
func (v *LocatorValidator) validateLocatorsSpecific(parts []string) error {
	// parts[0] = "", parts[1] = "locators", parts[2:len-2] = locator path components, parts[len-2] = cid, parts[len-1] = peer_id
	// Enhanced format: /locators/<locator_type>/<cid>/<peer_id>
	if len(parts) < types.MinLabelKeyParts {
		return errors.New("locators key must have format: /locators/<locator_type>/<cid>/<peer_id>")
	}

	// Extract locator type (everything between "locators" and CID)
	locatorParts := parts[2 : len(parts)-2] // Exclude CID and PeerID
	if len(locatorParts) == 0 {
		return errors.New("locator type cannot be empty")
	}

	// Validate that none of the locator path components are empty
	for i, part := range locatorParts {
		if part == "" {
			return errors.New("locator path component cannot be empty at position " + strconv.Itoa(i+1))
		}
	}

	// Future: validate against supported locator types
	// locatorType := strings.Join(locatorParts, "/")
	// if !v.isValidLocatorType(locatorType) {
	//     return errors.New("invalid locator type: " + locatorType)
	// }

	return nil
}

// Select chooses between multiple values for locators records.
func (v *LocatorValidator) Select(key string, values [][]byte) (int, error) {
	return v.selectFirstValid(key, values, v.Validate)
}

// CreateLabelValidators creates separate validators for each label namespace.
func CreateLabelValidators() map[string]record.Validator {
	return map[string]record.Validator{
		types.LabelTypeSkill.String():   &SkillValidator{},
		types.LabelTypeDomain.String():  &DomainValidator{},
		types.LabelTypeModule.String():  &ModuleValidator{},
		types.LabelTypeLocator.String(): &LocatorValidator{},
	}
}

// ValidateLabelKey validates a label key format before storing in DHT.
func ValidateLabelKey(key string) error {
	parts := strings.Split(key, "/")
	if len(parts) < types.MinLabelKeyParts {
		return errors.New("invalid key format: expected /<namespace>/<label_path>/<cid>")
	}

	namespace := parts[1]
	if _, valid := types.ParseLabelType(namespace); !valid {
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

// ExtractCIDFromLabelKey extracts CID from enhanced label key format.
// Example: "/skills/golang/CID123/Peer1" → "CID123", nil.
func ExtractCIDFromLabelKey(labelKey string) (string, error) {
	parts := strings.Split(labelKey, "/")
	if len(parts) < types.MinLabelKeyParts {
		return "", errors.New("invalid enhanced key format: expected /<namespace>/<label_path>/<cid>/<peer_id>")
	}

	// Validate it's a proper label key
	if !IsValidLabelKey(labelKey) {
		return "", errors.New("invalid namespace in label key")
	}

	// Extract and validate CID (second to last part)
	cidStr := parts[len(parts)-2]
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
