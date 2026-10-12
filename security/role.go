package security

import (
	"slices"
)

type Securable string

func (s Securable) String() string {
	return string(s)
}

type Role struct {
	Name       string      `json:"name"`
	Securables []Securable `json:"securables"`
}

func (r Role) Is(role *Role) bool {
	return r.Name == role.Name
}

func (r Role) Has(securable Securable) bool {
	return len(r.Securables) > 0 &&
		slices.Contains(r.Securables, securable)
}
