// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import "github.com/agntcy/dir/cli/presenter"

var opts = &options{}

type options struct {
	Limit  uint32
	Offset uint32

	// Direct field flags (consistent with routing search)
	Names       []string
	Versions    []string
	SkillIDs    []string
	SkillNames  []string
	Locators    []string
	Modules     []string
	DomainIDs   []string
	DomainNames []string
}

func init() {
	flags := Command.Flags()

	flags.Uint32Var(&opts.Limit, "limit", 100, "Maximum number of results to return (default: 100)") //nolint:mnd
	flags.Uint32Var(&opts.Offset, "offset", 0, "Pagination offset (default: 0)")

	// Direct field flags
	flags.StringArrayVar(&opts.Names, "name", nil, "Search for records with specific name (can be repeated)")
	flags.StringArrayVar(&opts.Versions, "version", nil, "Search for records with specific version (can be repeated)")
	flags.StringArrayVar(&opts.SkillIDs, "skill-id", nil, "Search for records with specific skill ID (can be repeated)")
	flags.StringArrayVar(&opts.SkillNames, "skill", nil, "Search for records with specific skill name (can be repeated)")
	flags.StringArrayVar(&opts.Locators, "locator", nil, "Search for records with specific locator type (can be repeated)")
	flags.StringArrayVar(&opts.Modules, "module", nil, "Search for records with specific module (can be repeated)")
	flags.StringArrayVar(&opts.DomainIDs, "domain-id", nil, "Search for records with specific domain ID (can be repeated)")
	flags.StringArrayVar(&opts.DomainNames, "domain", nil, "Search for records with specific domain name (can be repeated)")

	// Add examples in flag help
	flags.Lookup("name").Usage = "Search for records with specific name (e.g., --name 'my-agent' --name 'web-*')"
	flags.Lookup("version").Usage = "Search for records with specific version (e.g., --version 'v1.0.0' --version 'v1.*')"
	flags.Lookup("skill-id").Usage = "Search for records with specific skill ID (e.g., --skill-id '10201')"
	flags.Lookup("skill").Usage = "Search for records with specific skill name (e.g., --skill 'natural_language_processing' --skill 'audio')"
	flags.Lookup("locator").Usage = "Search for records with specific locator type (e.g., --locator 'docker-image')"
	flags.Lookup("module").Usage = "Search for records with specific module (e.g., --module 'runtime/language')"
	flags.Lookup("domain-id").Usage = "Search for records with specific domain ID (e.g., --domain-id '604')"
	flags.Lookup("domain").Usage = "Search for records with specific domain name (e.g., --domain '*education*' --domain 'healthcare/*')"

	// Add output format flags
	presenter.AddOutputFlags(Command)
}
