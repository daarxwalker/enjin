package abuse

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/daarxwalker/enjin/httpx"
)

const (
	botNamespace     = "bot"
	botTimeout       = time.Second
	botCacheTTL      = 24 * time.Hour
	botErrorTTL      = 5 * time.Minute
	botVerifiedValue = "1"
)

var (
	botUserAgentMarkers = []string{
		"googlebot",
		"adsbot-google",
		"google-inspectiontool",
		"storebot-google",
		"googleother",
		"bingbot",
	}
	botHostSuffixes = []string{
		".googlebot.com",
		".google.com",
		".search.msn.com",
	}
)

type BotVerifier struct {
	store    Store
	resolver resolver
	timeout  time.Duration
}

func NewBotVerifier(store Store, options ...BotVerifierOption) *BotVerifier {
	v := &BotVerifier{
		store:    store,
		resolver: net.DefaultResolver,
		timeout:  botTimeout,
	}
	for _, option := range options {
		option(v)
	}
	return v
}

func (v *BotVerifier) verifyBot(r *http.Request, addr netip.Addr, now time.Time) bool {
	if !v.claimsBot(r.UserAgent()) {
		return false
	}
	if httpx.IsFromCloudflare(r) && r.Header.Get("X-Verified-Bot") == botVerifiedValue {
		return true
	}
	key := Key{Namespace: botNamespace, Value: addr.String()}
	if good, found, err := v.store.LoadMark(key, now); err == nil && found {
		return good
	}
	good, ttl := v.confirmBotHost(r.Context(), addr)
	_ = v.store.SaveMark(key, good, ttl, now)
	return good
}

func (v *BotVerifier) claimsBot(userAgent string) bool {
	userAgent = strings.ToLower(userAgent)
	return slices.ContainsFunc(botUserAgentMarkers, func(marker string) bool {
		return strings.Contains(userAgent, marker)
	})
}

func (v *BotVerifier) confirmBotHost(ctx context.Context, addr netip.Addr) (bool, time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()
	names, err := v.resolver.LookupAddr(ctx, addr.String())
	if err != nil {
		return false, v.calcLookupFailureTTL(err)
	}
	for _, name := range names {
		host := strings.TrimSuffix(strings.ToLower(name), ".")
		if !v.isAllowedHost(host) {
			continue
		}
		ips, lookupErr := v.resolver.LookupIPAddr(ctx, host)
		if lookupErr != nil {
			return false, v.calcLookupFailureTTL(lookupErr)
		}
		for _, ip := range ips {
			if resolved, ok := netip.AddrFromSlice(ip.IP); ok && resolved.Unmap() == addr {
				return true, botCacheTTL
			}
		}
	}
	return false, botCacheTTL
}

func (v *BotVerifier) calcLookupFailureTTL(err error) time.Duration {
	if dnsErr, ok := errors.AsType[*net.DNSError](err); ok && dnsErr.IsNotFound {
		return botCacheTTL
	}
	return botErrorTTL
}

func (v *BotVerifier) isAllowedHost(host string) bool {
	return slices.ContainsFunc(botHostSuffixes, func(suffix string) bool {
		return strings.HasSuffix(host, suffix)
	})
}
