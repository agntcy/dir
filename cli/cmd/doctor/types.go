// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

type checkStatus string

const (
	statusPass checkStatus = "pass"
	statusFail checkStatus = "fail"
	statusWarn checkStatus = "warn"
	statusSkip checkStatus = "skip"
)

type checkResult struct {
	Name    string            `json:"name"`
	Status  checkStatus       `json:"status"`
	Message string            `json:"message"`
	Elapsed string            `json:"elapsed,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

type checkSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Warned  int `json:"warned"`
	Skipped int `json:"skipped"`
}

type configInfo struct {
	Context        string   `json:"context,omitempty"`
	ContextSource  string   `json:"context_source,omitempty"`
	ConfigPath     string   `json:"config_path,omitempty"`
	ServerAddress  string   `json:"server_address,omitempty"`
	AuthMode       string   `json:"auth_mode,omitempty"`
	TLSSkipVerify  bool     `json:"tls_skip_verify,omitempty"`
	BootstrapPeers int      `json:"bootstrap_peers"`
	Warnings       []string `json:"warnings,omitempty"`
}

type doctorOutput struct {
	Config  configInfo    `json:"config"`
	Summary checkSummary  `json:"summary"`
	Results []checkResult `json:"results"`
}

func skipped(name string, message string, details map[string]string) checkResult {
	return checkResult{
		Name:    name,
		Status:  statusSkip,
		Message: message,
		Details: details,
	}
}

func summarize(results []checkResult) checkSummary {
	summary := checkSummary{
		Total: len(results),
	}
	for _, result := range results {
		switch result.Status {
		case statusPass:
			summary.Passed++
		case statusFail:
			summary.Failed++
		case statusWarn:
			summary.Warned++
		case statusSkip:
			summary.Skipped++
		}
	}

	return summary
}

func hasFailure(results []checkResult) bool {
	for _, result := range results {
		if result.Status == statusFail {
			return true
		}
	}

	return false
}
