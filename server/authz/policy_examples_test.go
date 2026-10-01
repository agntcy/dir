// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	// Register every service a policy can name.
	_ "github.com/agntcy/dir/api/catalog/v1"
	_ "github.com/agntcy/dir/api/events/v1"
	_ "github.com/agntcy/dir/api/naming/v1"
	_ "github.com/agntcy/dir/api/policy/v1"
	_ "github.com/agntcy/dir/api/routing/v1"
	_ "github.com/agntcy/dir/api/runtime/v1"
	_ "github.com/agntcy/dir/api/search/v1"
	_ "github.com/agntcy/dir/api/sign/v1"
	_ "github.com/agntcy/dir/api/store/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// policyLine matches a policy naming one method, or every method of a
// service: "p,<subject>,/<package>.<Service>/<Method or *>".
var policyLine = regexp.MustCompile(`^\s*#?\s*p,\s*[^,\s]+,\s*/([A-Za-z0-9_.]+)/([A-Za-z0-9_]+|\*)\s*$`)

// Every policy the charts ship or the docs show must name a method the server
// serves: a misspelt one matches nothing, so it silently denies what it was
// meant to allow.
func TestPolicyExamples_NameServedMethods(t *testing.T) {
	t.Parallel()

	checked := 0

	for _, path := range policyFiles(t, "../../install", "../../docs", ".") {
		content, err := os.ReadFile(path)
		require.NoError(t, err)

		for i, line := range strings.Split(string(content), "\n") {
			match := policyLine.FindStringSubmatch(line)
			if match == nil {
				continue
			}

			checked++

			assert.Truef(t, isServed(match[1], match[2]), "%s:%d names /%s/%s, which the server does not serve", path, i+1, match[1], match[2])
		}
	}

	require.NotZero(t, checked, "no policies found")
}

// policyFiles lists the files under roots that can hold policies.
func policyFiles(t *testing.T, roots ...string) []string {
	t.Helper()

	var paths []string

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !entry.IsDir() && hasPolicyExtension(path) {
				paths = append(paths, path)
			}

			return nil
		})
		require.NoError(t, err)
	}

	return paths
}

func hasPolicyExtension(path string) bool {
	switch filepath.Ext(path) {
	case ".csv", ".md", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

// isServed reports whether service has method, or has any method when method
// is "*".
func isServed(service, method string) bool {
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(service))
	if err != nil {
		return false
	}

	serviceDescriptor, ok := descriptor.(protoreflect.ServiceDescriptor)
	if !ok {
		return false
	}

	if method == "*" {
		return serviceDescriptor.Methods().Len() > 0
	}

	return serviceDescriptor.Methods().ByName(protoreflect.Name(method)) != nil
}
