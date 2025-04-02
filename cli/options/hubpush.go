package options

type HubPushOptions struct {
	*HubOptions
	*PushOptions
}

func NewHubPushOptions(hubOptions *HubOptions, pushOptions *PushOptions) *HubPushOptions {
	return &HubPushOptions{
		HubOptions:  hubOptions,
		PushOptions: pushOptions,
	}
}
