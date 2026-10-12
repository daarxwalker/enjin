package abuse

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/netip"
	"strings"
)

type countedByKind uint8

const (
	countByIP countedByKind = iota
	countByClientID
	countByUserID
	countByFormField
)

type CountedBy struct {
	formField string
	name      string
	kind      countedByKind
}

type Identity struct {
	Addr     netip.Addr
	IP       netip.Prefix
	IPKey    string
	ClientID string
	UserID   string
	GoodBot  bool
}

var (
	CountedByIP       = CountedBy{kind: countByIP, name: "ip"}
	CountedByClientID = CountedBy{kind: countByClientID, name: "client"}
	CountedByUserID   = CountedBy{kind: countByUserID, name: "user"}
)

func CreateCountedByFormField(formField string) CountedBy {
	return CountedBy{kind: countByFormField, name: "form-" + formField, formField: formField}
}

func (c CountedBy) readValue(r *http.Request, id Identity) string {
	switch c.kind {
	case countByIP:
		return id.IPKey
	case countByClientID:
		return id.ClientID
	case countByUserID:
		return id.UserID
	case countByFormField:
		return c.hashFormValue(strings.ToLower(strings.TrimSpace(r.FormValue(c.formField))))
	}
	return ""
}

func (c CountedBy) hashFormValue(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}
