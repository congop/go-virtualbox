package virtualbox

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
)

// DHCP server info.
type DHCP struct {
	NetworkName string
	IPv4        net.IPNet
	LowerIP     net.IP
	UpperIP     net.IP
	Enabled     bool
}

func (dhcp DHCP) String() string {
	return fmt.Sprintf(
		"DHCP[%s, net=%s, start=%s stop=%s, enable=%t]",
		dhcp.NetworkName, dhcp.IPv4.String(), dhcp.LowerIP.String(), dhcp.LowerIP.String(), dhcp.Enabled)
}

func addDHCP(kind, name string, d DHCP) error {
	args := []string{"dhcpserver", "add",
		kind, name,
		"--ip", d.IPv4.IP.String(),
		"--netmask", net.IP(d.IPv4.Mask).String(),
		"--lowerip", d.LowerIP.String(),
		"--upperip", d.UpperIP.String(),
	}
	if d.Enabled {
		args = append(args, "--enable")
	} else {
		args = append(args, "--disable")
	}
	return Manage().run(args...)
}

// AddInternalDHCP adds a DHCP server to an internal network.
func AddInternalDHCP(netname string, d DHCP) error {
	return addDHCP("--netname", netname, d)
}

// AddHostonlyDHCP adds a DHCP server to a host-only network.
func AddHostonlyDHCP(ifname string, d DHCP) error {
	return addDHCP("--ifname", ifname, d)
}

// DHCPs gets all DHCP server settings in a map keyed by DHCP.NetworkName.
func DHCPs() (map[string]*DHCP, error) {
	out, err := Manage().runOut("list", "dhcpservers")
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(strings.NewReader(out))
	m := map[string]*DHCP{}
	dhcp := &DHCP{}
	for s.Scan() {
		line := s.Text()
		if line == "" {
			dhcp = &DHCP{}
			continue
		}
		res := reColonLine.FindStringSubmatch(line)
		if res == nil {
			continue
		}
		// output has change with version changes
		// - lowerIPAd.. /uppperIpAd.. now starting with upper case letter
		// - IP -> Dhcpd IP
		// so solution: using lowercase key for comparison
		switch key, val := strings.ToLower(res[1]), res[2]; key {
		case "networkname":
			dhcp.NetworkName = val
			if _, alreadyIn := m[dhcp.NetworkName]; alreadyIn {
				return nil, errors.Errorf(
					"DHCPs -- illegal state, dhcp server already parse: "+
						"\n\tnetworkname=%s \n\talready-parsed=%s \n\tout=%s",
					dhcp.NetworkName, maps.Keys(m), string(out))
			}
			// saving here so that we do not miss the last entry in case it is not
			// followed by an empty line
			m[dhcp.NetworkName] = dhcp
		case "ip", "dhcpd ip":
			dhcp.IPv4.IP = net.ParseIP(val).To4()
		case "upperipaddress":
			dhcp.UpperIP = net.ParseIP(val).To4()
		case "loweripaddress":
			dhcp.LowerIP = net.ParseIP(val).To4()
		case "networkmask":
			dhcp.IPv4.Mask = ParseIPv4Mask(val)
		case "enabled":
			dhcp.Enabled = (val == stringYes)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return m, nil
}
