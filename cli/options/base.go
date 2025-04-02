package options

import (
	"errors"

	"github.com/spf13/cobra"
)

type RegisterFn func(cmd *cobra.Command) error
type CompleteFn func()

type BaseOption struct {
	isCompleted  bool
	isRegistered bool
	err          error

	registerFns []RegisterFn
	completeFns []CompleteFn
}

func (o *BaseOption) addErr(err error) {
	if err == nil {
		o.err = err
	} else {
		o.err = errors.Join(o.err, err)
	}
}

func (o *BaseOption) CheckError() error {
	return o.err
}

func (o *BaseOption) Register(cmd *cobra.Command) {
	if o.isRegistered {
		return
	}
	defer func() {
		o.isRegistered = true
	}()
	for _, fn := range o.registerFns {
		if err := fn(cmd); err != nil {
			o.addErr(err)
			return
		}
	}
}

func (o *BaseOption) Complete() {
	if o.isCompleted {
		return
	}
	defer func() {
		o.isCompleted = true
	}()
	for _, fn := range o.completeFns {
		fn()
	}
}

func (o *BaseOption) AddRegisterFns(fns []RegisterFn) {
	o.registerFns = append(o.registerFns, fns...)
}

func (o *BaseOption) AddCompleteFn(fns []CompleteFn) {
	o.completeFns = append(o.completeFns, fns...)
}
