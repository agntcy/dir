package options

type HubPullOptions struct {
	*HubOptions
}

func NewHubPullOptions(hubOptions *HubOptions) *HubPullOptions {
	return &HubPullOptions{
		HubOptions: hubOptions,
	}
}
