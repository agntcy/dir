package options

import (
	"errors"
)

type RegisterFn func() error
type CompleteFn func()

type BaseOption struct {
	isCompleted  bool
	isRegistered bool
	err          error

	registerFns []RegisterFn
}

func NewBaseOption() *BaseOption {
	return &BaseOption{}
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

func (o *BaseOption) Register() error {
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

func (o *BaseOption) AddRegisterFns(fns RegisterFn) {
	o.registerFns = append(o.registerFns, fns)
}
