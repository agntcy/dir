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
	isCompleted  bool

	registerFns []RegisterFn
	completeFns []CompleteFn
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
		return nil
	}

	for _, fn := range o.registerFns {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
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

func (o *BaseOption) AddRegisterFns(fn RegisterFn) {
	o.registerFns = append(o.registerFns, fn)
}

func (o *BaseOption) AddCompleteFns(fn CompleteFn) {
	o.completeFns = append(o.completeFns, fn)
}
