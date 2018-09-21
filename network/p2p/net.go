package p2p

import (
	"net"
)

// Netlist is a list of IP networks.
type Netlist []net.IPNet

// Contains reports whether the given IP is contained in the list.
func (l *Netlist) Contains(ip net.IP) bool {
	if l == nil {
		return false
	}
	for _, net := range *l {
		if net.Contains(ip) {
			return true
		}
	}
	return false
}
