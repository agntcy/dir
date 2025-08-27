package casbin

import (
	"errors"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

// DirPolicyAdapter implements the casbin.Adapter
// interface to load policy from a directory.
type Adapter struct {
	policyContent string
}

func NewAdapter() (persist.Adapter, error) {
	return &Adapter{}, nil
}

// AddPolicy adds a policy rule to the storage.
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	return errors.New("not implemented")
}

// RemovePolicy removes a policy rule from the storage.
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return errors.New("not implemented")
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return errors.New("not implemented")
}

// LoadPolicy implements persist.Adapter.
func (a *Adapter) LoadPolicy(model model.Model) error {
	panic("unimplemented")
}

// SavePolicy implements persist.Adapter.
func (a *Adapter) SavePolicy(model model.Model) error {
	panic("unimplemented")
}
