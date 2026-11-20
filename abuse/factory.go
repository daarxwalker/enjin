package abuse

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/daarxwalker/enjin/client"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/session"
)

const (
	ipv4PrefixBits = 32
	ipv6PrefixBits = 64
)

func createIdentity(r *http.Request) Identity {
	id := Identity{ClientID: client.IDFrom(r.Context())}
	if profile := session.From(r.Context()); profile != nil {
		id.UserID = profile.ID
	}
	addr, err := netip.ParseAddr(httpx.ClientIP(r))
	if err != nil {
		id.IPKey = r.RemoteAddr
		return id
	}
	id.Addr = addr.Unmap()
	addr = id.Addr
	bits := ipv4PrefixBits
	if addr.Is6() {
		bits = ipv6PrefixBits
	}
	prefix, err := addr.Prefix(bits)
	if err != nil {
		id.IPKey = addr.String()
		return id
	}
	id.IP = prefix
	id.IPKey = prefix.String()
	return id
}

func createMemberSalt() uint32 {
	var salt [4]byte
	// Since Go 1.24 crypto/rand.Read crashes the program instead of returning
	// an error, so this branch is unreachable.
	if _, err := rand.Read(salt[:]); err != nil {
		panic(fmt.Errorf("read random member salt failed: %w", err))
	}
	return binary.LittleEndian.Uint32(salt[:])
}
