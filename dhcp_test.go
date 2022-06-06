package virtualbox

import (
	"net"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func TestDHCPs(t *testing.T) {
	Setup(t)
	defer Teardown()

	if ManageMock != nil {
		listDhcpServersOut := ReadTestData("vboxmanage-list-dhcpservers-1.out")
		gomock.InOrder(
			ManageMock.EXPECT().runOut("list", "dhcpservers").Return(listDhcpServersOut, nil).Times(1),
		)
	}
	m, err := DHCPs()
	if err != nil {
		t.Fatal(err)
	}

	require.Lenf(t, m, 2, "must have been 2 dhcp servers available ")
	keys := maps.Keys(m)
	slices.Sort(keys)
	servers := make([]DHCP, 0, len(keys))
	for _, k := range keys {
		v := m[k]
		servers = append(servers, *v)
	}

	expectedServers := []DHCP{
		{
			NetworkName: "HostInterfaceNetworking-VirtualBox Host-Only Ethernet Adapter",
			IPv4:        mustCidrKeepUnmaskIp(t, "192.168.56.100/24"),
			LowerIP:     mustParseIp(t, "192.168.56.101"),
			UpperIP:     mustParseIp(t, "192.168.56.254"),
			Enabled:     false,
		},

		{
			NetworkName: "HostInterfaceNetworking-vboxnet5",
			IPv4:        mustCidrKeepUnmaskIp(t, "192.168.61.1/24"),
			LowerIP:     mustParseIp(t, "192.168.61.50"),
			UpperIP:     mustParseIp(t, "192.168.61.200"),
			Enabled:     true,
		},
	}
	require.Equalf(t, expectedServers, servers,
		"servers should match\n\texpected=%s \n\tactual=%s", expectedServers, servers)

}

func mustParseIp(t *testing.T, ipStr string) net.IP {
	ip := net.ParseIP(ipStr)
	//require.NoErrorf(t, err, "fail to parse cidr:%s", err)
	return ip.To4()
}

func mustCidrKeepUnmaskIp(t *testing.T, cidrStr string) net.IPNet {

	ip, cidr, err := net.ParseCIDR(cidrStr)
	cidr.IP = ip
	if !strings.Contains(cidrStr, ":") {
		// ipv4
		cidr.IP = cidr.IP.To4()

	}
	require.NoErrorf(t, err, "fail to parse cidr:%s", err)
	return *cidr
}
