package oasf

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/discovery/pkg/types"
)

// processor probes workloads for OASF data.
type processor struct {
	timeout  time.Duration
	client   *client.Client
	labelKey string
}

// NewProcessor creates a new OASF processor.
func NewProcessor(cfg Config) (types.WorkloadProcessor, error) {
	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Create Dir client
	dirClient, err := client.New(ctx, client.WithEnvConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create Dir client: %v", err)
	}

	// Return processor
	return &processor{
		timeout:  cfg.Timeout,
		client:   dirClient,
		labelKey: cfg.LabelKey,
	}, nil
}

// Name returns the processor name.
func (p *processor) Name() string {
	return "oasf"
}

// ShouldProcess returns whether to process the workload.
func (p *processor) ShouldProcess(workload *types.Workload) bool {
	// If workload does not have a label key, skip it
	if _, ok := workload.Labels[p.labelKey]; !ok {
		return false
	}

	return true
}

// Process probes health endpoints on the workload.
func (p *processor) Process(ctx context.Context, workload *types.Workload) (interface{}, error) {
	// Get the name of the OASF record from the workload labels
	recordFQDN := workload.Labels[p.labelKey]
	nameParts := strings.SplitN(recordFQDN, ":", 2)
	if len(nameParts) != 2 {
		return nil, fmt.Errorf("invalid OASF record format in label %s: %s", p.labelKey, recordFQDN)
	}

	// Fetch the OASF record
	resolve, err := p.client.Resolve(ctx, nameParts[0], nameParts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OASF record %s: %v", recordFQDN, err)
	}

	// Get the first info
	if len(resolve.Records) == 0 {
		return nil, fmt.Errorf("no OASF info found for record %s", recordFQDN)
	}
	recordRef := resolve.Records[0]

	// Pull the full record data
	record, err := p.client.Pull(ctx, &corev1.RecordRef{Cid: recordRef.Cid})
	if err != nil {
		return nil, fmt.Errorf("failed to pull OASF record %s: %v", recordFQDN, err)
	}

	// Convert the record data to a map
	recordData := record.Data.AsMap()

	// Check if its signed
	signed, err := p.client.Verify(ctx, &signv1.VerifyRequest{RecordRef: &corev1.RecordRef{Cid: recordRef.Cid}})
	if err != nil {
		return nil, fmt.Errorf("failed to verify OASF record %s: %v", recordFQDN, err)
	}

	// Check if its name is verified
	verified, err := p.client.GetVerificationInfo(ctx, recordRef.Cid)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification info for OASF record %s: %v", recordFQDN, err)
	}

	log.Printf("[oasf] %s scraped successfully", workload.Name)

	// Return the record data along with its validity
	return map[string]any{
		"record":   recordData,
		"signed":   signed.Success,
		"verified": verified.Verified,
	}, nil
}
