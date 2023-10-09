package virtualbox

import (
	"bufio"
	"net"
	"strings"
)

// A NATNet defines a NAT network.
type NATNet struct {
	Name    string
	IPv4    net.IPNet
	IPv6    net.IPNet
	DHCP    bool
	Enabled bool
}

// NATNets gets all NAT networks in a  map keyed by NATNet.Name.
func NATNets() (map[string]NATNet, error) {

	// VBoxManage list natnets --long
	// Virtualbox 6.xxx
	// NetworkName:    sayanat
	// IP:             10.12.0.1
	// Network:        10.12.0.0/24
	// IPv6 Enabled:   Yes
	// IPv6 Prefix:    fd17:625c:f037:2::/64
	// DHCP Enabled:   Yes
	// Enabled:        Yes
	// loopback mappings (ipv4)
	//         127.0.0.1=2
	//
	// Virtualbox 7.0
	// Name:         sayanat
	// Enabled:      Yes
	// Network:      10.12.0.0/24
	// Gateway:      10.12.0.1
	// DHCP Server:  Yes
	// IPv6:         Yes
	// IPv6 Prefix:  fd17:625c:f037:2::/64
	// IPv6 Default: No
	// loopback mappings (ipv4)
	// 		127.0.0.1=2

	out, err := Manage().runOut("list", "natnets")
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(strings.NewReader(out))
	m := map[string]NATNet{}
	n := NATNet{}
	for s.Scan() {
		line := s.Text()
		if line == "" {
			m[n.Name] = n
			n = NATNet{}
			continue
		}
		res := reColonLine.FindStringSubmatch(line)
		if res == nil {
			continue
		}
		switch key, val := res[1], res[2]; key {
		case "Name", "NetworkName":
			n.Name = val
		case "IP", "Gateway":
			n.IPv4.IP = net.ParseIP(val)
		case "Network":
			_, ipnet, err := net.ParseCIDR(val)
			if err != nil {
				return nil, err
			}
			n.IPv4.Mask = ipnet.Mask
		case "IPv6 Prefix":
			// TODO: IPv6 CIDR parsing works fine on macOS, check on Windows
			// if val == "" {
			// 	continue
			// }
			// l, err := strconv.ParseUint(val, 10, 7)
			// if err != nil {
			// 	return nil, err
			// }
			// n.IPv6.Mask = net.CIDRMask(int(l), net.IPv6len*8)
			_, ipnet, err := net.ParseCIDR(val)
			if err != nil {
				return nil, err
			}
			n.IPv6.Mask = ipnet.Mask
		case "DHCP Enabled":
			n.DHCP = (val == stringYes)
		case "Enabled":
			n.Enabled = (val == stringYes)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return m, nil
}
