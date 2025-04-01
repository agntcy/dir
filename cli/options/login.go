package options

type LoginOptions struct {
	*HubOptions
}

func NewLoginOptions(hubOptions *HubOptions) *LoginOptions {
	return &LoginOptions{
		HubOptions: hubOptions,
	}
}
