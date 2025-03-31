package options

import (
	"fmt"

	"github.com/spf13/cobra"
)

type BaseOption struct {
	err error
}

func (o *BaseOption) AddError(err error) {
	if o.err == nil {
		o.err = err
	} else {
		o.err = fmt.Errorf("%w: %w", o.err, err)
	}
}

func (o *BaseOption) CheckError() error {
	return o.err
}

func (o *BaseOption) Register(cmd *cobra.Command) {
	panic("implement me")
}

func (o *BaseOption) Complete() {
	panic("implement me")
}
