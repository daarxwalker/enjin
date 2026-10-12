package session

import (
	"github.com/daarxwalker/enjin/security"
)

type Profile struct {
	Role   *security.Role    `json:"role"`
	Values map[string]string `json:"values"`
	ID     string            `json:"id"`
	Key    string            `json:"key"`
	Token  string            `json:"token"`
}

type cacheProfile struct {
	Values map[string]string `json:"values"`
	ID     string            `json:"id"`
	Key    string            `json:"key"`
	Role   string            `json:"role"`
	Token  string            `json:"token"`
}
