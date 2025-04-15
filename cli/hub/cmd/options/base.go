// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package options

type (
	RegisterFn func() error
	CompleteFn func()
)

type BaseOption struct {
	isRegistered bool
	isCompleted  bool

	registerFns []RegisterFn
	completeFns []CompleteFn
}

func NewBaseOption() *BaseOption {
	return &BaseOption{}
}

func (o *BaseOption) Register() error {
	if o.isRegistered {
		return nil
	}

	defer func() {
		o.isRegistered = true
	}()

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

func (o *BaseOption) AddRegisterFn(fn RegisterFn) {
	if fn == nil {
		return
	}

	o.registerFns = append(o.registerFns, fn)
}

func (o *BaseOption) AddCompleteFn(fn CompleteFn) {
	if fn == nil {
		return
	}

	o.completeFns = append(o.completeFns, fn)
}
