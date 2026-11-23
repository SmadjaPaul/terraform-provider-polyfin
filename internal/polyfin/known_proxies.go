package polyfin

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"
)

// MaxKnownProxies mirrors upstream accounts.MaxKnownProxies: addresses,
// networks and host names together.
const MaxKnownProxies = 64

// CanonicalKnownProxies returns entries as Polyfin stores and answers them
// (upstream accounts.ParseKnownProxies then FormatKnownProxies): addresses and
// networks first, a network without its host bits, a single address without
// its prefix length, IPv4-mapped IPv6 as IPv4, then host names in lower case,
// each entry once, in the order given. Comparing canonical forms keeps the
// plan a no-op when Polyfin reorders or normalizes what was written.
func CanonicalKnownProxies(entries []string) ([]string, error) {
	if len(entries) > MaxKnownProxies {
		return nil, errors.New("known proxies hold at most 64 entries")
	}
	networks := []netip.Prefix{}
	names := []string{}
	for _, raw := range entries {
		entry := strings.TrimSpace(raw)
		if entry != raw || entry == "" {
			return nil, errors.New("known proxy entries must be nonempty and without surrounding spaces")
		}
		network, err := netip.ParsePrefix(entry)
		if err != nil {
			address, addrErr := netip.ParseAddr(entry)
			if addrErr != nil {
				if !validProxyHostName(entry) {
					return nil, errors.New("known proxy entry is no IP address, CIDR network or host name")
				}
				if name := strings.ToLower(entry); !slices.Contains(names, name) {
					names = append(names, name)
				}
				continue
			}
			if address.Zone() != "" {
				return nil, errors.New("known proxy address must not carry an IPv6 zone")
			}
			network = netip.PrefixFrom(address, address.BitLen())
		}
		if address := network.Addr(); address.Is4In6() {
			if network.Bits() < 96 {
				return nil, errors.New("invalid IPv4-mapped known proxy network")
			}
			network = netip.PrefixFrom(address.Unmap(), network.Bits()-96)
		}
		if network = network.Masked(); !slices.Contains(networks, network) {
			networks = append(networks, network)
		}
	}
	out := make([]string, 0, len(networks)+len(names))
	for _, network := range networks {
		if network.IsSingleIP() {
			out = append(out, network.Addr().String())
		} else {
			out = append(out, network.String())
		}
	}
	return append(out, names...), nil
}

// validProxyHostName mirrors upstream accounts.validHostName.
func validProxyHostName(name string) bool {
	name = strings.TrimSuffix(name, ".")
	if name == "" || len(name) > 253 {
		return false
	}
	labels := strings.Split(name, ".")
	for _, label := range labels {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range []byte(label) {
			if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '-' || c == '_') {
				return false
			}
		}
	}
	last := labels[len(labels)-1]
	for _, c := range []byte(last) {
		if c < '0' || c > '9' {
			return true
		}
	}
	return false
}

// GetKnownProxies reads Settings > Security > Known proxies (Polyfin 0.80+).
func (c *Client) GetKnownProxies(ctx context.Context) ([]string, error) {
	if err := c.CheckAdmin(ctx); err != nil {
		return nil, err
	}
	var result struct {
		KnownProxies *[]string `json:"knownProxies"`
	}
	if _, err := c.request(ctx, "GET", "/admin/api/settings", nil, &result); err != nil {
		return nil, err
	}
	if result.KnownProxies == nil {
		return nil, errors.New("Polyfin settings missing knownProxies (requires a Polyfin release with Known proxies)")
	}
	return *result.KnownProxies, nil
}

// SetKnownProxies sends only knownProxies: upstream's settings PUT is
// partial, so every other setting keeps its value. The value is read back
// and compared in canonical form.
func (c *Client) SetKnownProxies(ctx context.Context, entries []string) error {
	want, err := CanonicalKnownProxies(entries)
	if err != nil {
		return err
	}
	if err := c.CheckAdmin(ctx); err != nil {
		return err
	}
	if err := c.putSetting(ctx, map[string]any{"knownProxies": want}); err != nil {
		return err
	}
	current, err := c.GetKnownProxies(ctx)
	if err != nil {
		return err
	}
	got, err := CanonicalKnownProxies(current)
	if err != nil || !slices.Equal(got, want) {
		return fmt.Errorf("Polyfin knownProxies readback mismatch")
	}
	return nil
}
