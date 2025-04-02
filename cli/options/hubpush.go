package options

type HubPushOptions struct {
	*HubOptions
	*PushOptions
}

func NewHubPushOptions(hubOptions *HubOptions, pushOptions *PushOptions) *HubPushOptions {
	pushOptions.addErr(hubOptions.err)
	pushOptions.AddRegisterFns(hubOptions.registerFns)
	pushOptions.AddCompleteFn(hubOptions.completeFns)
	hubOptions.BaseOption = pushOptions.BaseOption
	return &HubPushOptions{
		HubOptions:  hubOptions,
		PushOptions: pushOptions,
	}
}
