// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package python

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	toml "github.com/pelletier/go-toml"
	"go.uber.org/multierr"

	"github.com/agntcy/dir/cli/builder/extensions/runtime/analyzer"
	"github.com/agntcy/dir/cli/builder/extensions/runtime/analyzer/utils/syft"
)

const (
	language = "python"
)

var (
	// no version found in pyproject.toml, Pipfile, or requirements.txt
	errNoVersion = errors.New("no version found in file")
)

type pythonAnalyzer struct {
	syft syft.Syft
}

func New() analyzer.Analyzer {
	return &pythonAnalyzer{
		syft: syft.Syft{},
	}
}

var SupportedAgentFrameworkPackages = []string{
	"crewai",
	"langchain",
	"langchain-ollama",
	"langgraph",
	"langchain-openai",
	"llama-deploy",
	"autogen-core",
}

func (a *pythonAnalyzer) SBOM(path string) (analyzer.SBOM, error) {
	return a.syft.SBOM(path, SupportedAgentFrameworkPackages)
}

func (a *pythonAnalyzer) RuntimeVersion(path string) (analyzer.RuntimeInfo, error) {
	return getRuntimeInfo(path)
}

// ExtractMinimalPythonVersion attempts to determine the minimal Python version required for a package.
func getRuntimeInfo(path string) (analyzer.RuntimeInfo, error) {
	// NOTE(msardara): here we *could* specify also an image name and the python runtime version would be given
	// by the image SBOM, but for now we are only looking at the source code
	ret, err := resolveFileSystemPath(path)
	if err == nil {
		ret.Language = language
	}

	return ret, err
}

func resolveFileSystemPath(path string) (analyzer.RuntimeInfo, error) {
	// the version
	version := ""
	var err error
	var errs []error

	// Check if path is a directory or a file
	fileInfo, err := os.Stat(path)
	if err != nil {
		return analyzer.RuntimeInfo{}, err
	}

	if !fileInfo.IsDir() {
		path = filepath.Dir(path)
	}

	// Check for pyproject.toml (Poetry)
	pyprojectPath := filepath.Join(path, "pyproject.toml")
	if _, err = os.Stat(pyprojectPath); err == nil {
		if version, err = parsePyprojectToml(pyprojectPath); err == nil {
			return analyzer.RuntimeInfo{
				Version: version,
			}, nil
		}
	}

	errs = append(errs, err)

	// Check for Pipfile (Pipenv)
	pipfilePath := filepath.Join(path, "Pipfile")
	if _, err = os.Stat(pipfilePath); err == nil {
		if version, err = parsePipfile(pipfilePath); err == nil {
			return analyzer.RuntimeInfo{
				Version: version,
			}, nil
		}
	}

	errs = append(errs, err)

	// Check for setup.py
	setuppyPath := filepath.Join(path, "setup.py")
	if _, err = os.Stat(setuppyPath); err == nil {
		if version, err = parseSetupPy(setuppyPath); err == nil {
			return analyzer.RuntimeInfo{
				Version: version,
			}, nil
		}
	}

	errs = append(errs, err)

	// Add more checks for other files/formats as needed
	return analyzer.RuntimeInfo{
		Version: version,
	}, multierr.Combine(errs...)
}

func parsePyprojectToml(path string) (string, error) {
	config, err := toml.LoadFile(path)
	if err != nil {
		return "", err
	}

	// Try first with the standard "requires-python" in the [project] section
	requires := config.Get("project.requires-python")
	if requires != nil {
		return requires.(string), nil
	}

	// Try with the "python" in the [tool.poetry.dependencies] section
	requires = config.Get("tool.poetry.dependencies.python")
	if requires != nil {
		return requires.(string), nil
	}

	// No luck
	return "", fmt.Errorf("%w: %s", errNoVersion, path)
}

func parsePipfile(path string) (string, error) {
	config, err := toml.LoadFile(path)
	if err != nil {
		return "", err
	}

	// Try with the "python_version" in the [requires] section
	requires := config.Get("requires.python_version")
	if requires != nil {
		return requires.(string), nil
	}

	// No luck
	return "", fmt.Errorf("%w: %s", errNoVersion, path)
}

// parse the setup.py file to find the python version in python_requires
func parseSetupPy(path string) (string, error) {
	// NOTE(msardara): this won't work if the version string is stored in a variable
	regexPattern := `python_requires\s*=\s*['"]([^'"]+)['"]`
	re := regexp.MustCompile(regexPattern)

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check for a match on each line
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			// The first submatch contains the version string
			return matches[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("%w: %s", errNoVersion, path)
}
