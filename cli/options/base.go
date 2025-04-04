// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

type (
	RegisterFn func() error
	CompleteFn func()
)

type BaseOption struct {
	err          error
	isRegistered bool

	registerFns []RegisterFn
}

func NewBaseOption() *BaseOption {
	return &BaseOption{}
}

func (o *BaseOption) CheckError() error {
	return o.err
}

func (o *BaseOption) Register() error {
	defer func() {
		o.isRegistered = true
	}()

	if o.isRegistered {
		for _, fn := range o.registerFns {
			if err := fn(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (o *BaseOption) AddRegisterFns(fns RegisterFn) {
	o.registerFns = append(o.registerFns, fns)
}
