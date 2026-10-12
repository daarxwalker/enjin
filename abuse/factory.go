package abuse

import (
	"crypto/rand"
	"encoding/binary"
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
	id.IP, _ = addr.Prefix(bits)
	id.IPKey = id.IP.String()
	return id
}

func createMemberSalt() uint32 {
	var salt [4]byte
	_, _ = rand.Read(salt[:])
	return binary.LittleEndian.Uint32(salt[:])
}
