package compare

import (
	"fmt"
	"math/big"
	"net/netip"
	"sort"
)

// Intervals are inclusive, disjoint, sorted, and family-specific. big.Int keeps
// /0 and IPv6 endpoints exact; no address in a network is enumerated.
type interval struct {
	lo, hi *big.Int
	bits   int
}

func clone(n *big.Int) *big.Int { return new(big.Int).Set(n) }
func number(a netip.Addr) *big.Int {
	if a.Is4() {
		b := a.As4()
		return new(big.Int).SetBytes(b[:])
	}
	b := a.As16()
	return new(big.Int).SetBytes(b[:])
}
func address(n *big.Int, bits int) string {
	if bits == 32 {
		var a [4]byte
		n.FillBytes(a[:])
		return netip.AddrFrom4(a).String()
	}
	var a [16]byte
	n.FillBytes(a[:])
	return netip.AddrFrom16(a).String()
}
func networks(items []string, required bool) ([]interval, error) {
	if len(items) == 0 {
		if required {
			return nil, nil
		}
		return []interval{{big.NewInt(0), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 32), big.NewInt(1)), 32}, {big.NewInt(0), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1)), 128}}, nil
	}
	if len(items) > 10000 {
		return nil, fmt.Errorf("access list exceeds 10000 entries")
	}
	var out []interval
	for i, s := range items {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			a, e := netip.ParseAddr(s)
			if e != nil || a.Zone() != "" {
				return nil, fmt.Errorf("access list entry %d must be an IP address or CIDR", i+1)
			}
			p = netip.PrefixFrom(a, a.BitLen())
		}
		if p.Addr().Is4In6() || p.Addr().Zone() != "" {
			return nil, fmt.Errorf("access list entry %d: mapped IPv6 and zone identifiers are unsupported", i+1)
		}
		p = p.Masked()
		bits := p.Addr().BitLen()
		lo := number(p.Addr())
		size := new(big.Int).Lsh(big.NewInt(1), uint(bits-p.Bits()))
		hi := new(big.Int).Sub(new(big.Int).Add(lo, size), big.NewInt(1))
		out = append(out, interval{lo, hi, bits})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].bits != out[j].bits {
			return out[i].bits < out[j].bits
		}
		return out[i].lo.Cmp(out[j].lo) < 0
	})
	var merged []interval
	for _, r := range out {
		if len(merged) == 0 {
			merged = append(merged, r)
			continue
		}
		last := &merged[len(merged)-1]
		next := new(big.Int).Add(last.hi, big.NewInt(1))
		if last.bits == r.bits && r.lo.Cmp(next) <= 0 {
			if r.hi.Cmp(last.hi) > 0 {
				last.hi = r.hi
			}
		} else {
			merged = append(merged, r)
		}
	}
	return merged, nil
}

// difference walks normalized sets once, producing the exact A \ B ranges.
func difference(a, b []interval) []interval {
	var out []interval
	j := 0
	for _, x := range a {
		cursor := clone(x.lo)
		for j < len(b) && (b[j].bits < x.bits || (b[j].bits == x.bits && b[j].hi.Cmp(cursor) < 0)) {
			j++
		}
		k := j
		for k < len(b) && b[k].bits == x.bits && b[k].lo.Cmp(x.hi) <= 0 {
			y := b[k]
			if y.lo.Cmp(cursor) > 0 {
				out = append(out, interval{clone(cursor), new(big.Int).Sub(y.lo, big.NewInt(1)), x.bits})
			}
			if y.hi.Cmp(x.hi) >= 0 {
				cursor = new(big.Int).Add(x.hi, big.NewInt(1))
				break
			}
			if y.hi.Cmp(cursor) >= 0 {
				cursor = new(big.Int).Add(y.hi, big.NewInt(1))
			}
			k++
		}
		if cursor.Cmp(x.hi) <= 0 {
			out = append(out, interval{cursor, clone(x.hi), x.bits})
		}
		j = k
	}
	return out
}

type NetworkDelta struct {
	CIDR      string `json:"cidr"`
	Witness   string `json:"witness"`
	Addresses string `json:"addresses"`
	Family    string `json:"family"`
}

func describe(rs []interval) []NetworkDelta {
	out := []NetworkDelta{}
	for _, r := range rs {
		cur := clone(r.lo)
		for cur.Cmp(r.hi) <= 0 {
			align := r.bits
			if cur.Sign() != 0 {
				align = int(cur.TrailingZeroBits())
			}
			remaining := new(big.Int).Add(new(big.Int).Sub(r.hi, cur), big.NewInt(1))
			power := remaining.BitLen() - 1
			if align < power {
				power = align
			}
			size := new(big.Int).Lsh(big.NewInt(1), uint(power))
			a := address(cur, r.bits)
			family := "IPv6"
			if r.bits == 32 {
				family = "IPv4"
			}
			out = append(out, NetworkDelta{fmt.Sprintf("%s/%d", a, r.bits-power), a, size.String(), family})
			cur.Add(cur, size)
		}
	}
	return out
}
