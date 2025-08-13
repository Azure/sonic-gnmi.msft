package ipinterfaces

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"golang.org/x/sys/unix"
)

// Helpers to build synthetic rtnetlink messages the same way the kernel would.
func newRTMNewLinkMsg(idx int32, flags uint32, name string, masterIdx int32) netlink.Message {
	// Build ifinfomsg header.
	hdr := make([]byte, unix.SizeofIfInfomsg)
	// ifi_index at bytes 4..8
	copy(hdr[4:8], nlenc.Uint32Bytes(uint32(idx)))
	// ifi_flags at bytes 8..12
	copy(hdr[8:12], nlenc.Uint32Bytes(flags))

	// Attributes: IFLA_IFNAME [+ IFLA_MASTER]
	attrs := []netlink.Attribute{
		{Type: unix.IFLA_IFNAME, Data: append([]byte(name), 0x00)}, // NUL-terminated name
	}
	if masterIdx != 0 {
		attrs = append(attrs, netlink.Attribute{Type: unix.IFLA_MASTER, Data: nlenc.Uint32Bytes(uint32(masterIdx))})
	}
	attrBytes, _ := netlink.MarshalAttributes(attrs)

	return netlink.Message{
		Header: netlink.Header{Type: unix.RTM_NEWLINK},
		Data:   append(hdr, attrBytes...),
	}
}

func newRTMNewAddrMsg(family uint8, ifidx int32, ipStr string, prefix int) netlink.Message {
	ifa := make([]byte, unix.SizeofIfAddrmsg)
	ifa[0] = family       // ifa_family
	ifa[1] = byte(prefix) // ifa_prefixlen
	copy(ifa[4:8], nlenc.Uint32Bytes(uint32(ifidx)))

	var ip net.IP
	if family == unix.AF_INET {
		ip = net.ParseIP(ipStr).To4()
	} else {
		ip = net.ParseIP(ipStr).To16()
	}
	if ip == nil {
		ip = net.IP{}
	}
	attrs := []netlink.Attribute{{Type: unix.IFA_LOCAL, Data: []byte(ip)}}
	attrBytes, _ := netlink.MarshalAttributes(attrs)

	return netlink.Message{
		Header: netlink.Header{Type: unix.RTM_NEWADDR},
		Data:   append(ifa, attrBytes...),
	}
}

func TestGetInterfacesInNamespace_IPv4_DefaultNS(t *testing.T) {
	// Prepare synthetic rtnetlink dumps
	linkMsgs := []netlink.Message{
		newRTMNewLinkMsg(1, unix.IFF_UP, "Ethernet1", 0),
		newRTMNewLinkMsg(2, unix.IFF_UP, "PortChannel1", 0),
	}
	addrMsgs := []netlink.Message{
		newRTMNewAddrMsg(unix.AF_INET, 1, "10.0.0.1", 31),
	}

	patches := gomonkey.ApplyFunc(unix.Open, func(path string, flags int, mode uint32) (int, error) {
		// Only original namespace should be opened in defaultNS
		if path == "/proc/self/ns/net" {
			if flags != unix.O_RDONLY {
				t.Fatalf("unix.Open flags=%d want O_RDONLY", flags)
			}
			return 100, nil
		}
		if strings.HasPrefix(path, "/var/run/netns/") {
			return -1, fmt.Errorf("unexpected target netns open in defaultNS: %s", path)
		}
		return -1, fmt.Errorf("unexpected open path: %s", path)
	})
	defer patches.Reset()

	patches.ApplyFunc(unix.Setns, func(fd int, nstype int) error { return nil })
	// Stub Close to no-op for fake fds
	patches.ApplyFunc(unix.Close, func(fd int) error { return nil })
	patches.ApplyFunc(netlink.Dial, func(family int, cfg *netlink.Config) (*netlink.Conn, error) { return &netlink.Conn{}, nil })
	patches.ApplyMethod(reflect.TypeOf(&netlink.Conn{}), "Close", func(_ *netlink.Conn) error { return nil })
	patches.ApplyMethod(reflect.TypeOf(&netlink.Conn{}), "Execute", func(_ *netlink.Conn, req netlink.Message) ([]netlink.Message, error) {
		switch req.Header.Type {
		case unix.RTM_GETLINK:
			return linkMsgs, nil
		case unix.RTM_GETADDR:
			return addrMsgs, nil
		default:
			return nil, nil
		}
	})

	ifs, err := getInterfacesInNamespace(defaultNamespace, AddressFamilyIPv4)
	if err != nil {
		t.Fatalf("getInterfacesInNamespace default ns failed: %v", err)
	}
	if len(ifs) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(ifs))
	}

	// Map by name for assertions independent of order
	byName := map[string]IPInterfaceDetail{}
	for _, i := range ifs {
		byName[i.Name] = i
	}

	eth1, ok := byName["Ethernet1"]
	if !ok {
		t.Fatalf("missing Ethernet1 in result")
	}
	if eth1.AdminStatus != "up" {
		t.Errorf("Ethernet1 admin status: want up, got %s", eth1.AdminStatus)
	}
	if len(eth1.IPAddresses) != 1 || eth1.IPAddresses[0].Address != "10.0.0.1/31" {
		t.Errorf("Ethernet1 IPs: want [10.0.0.1/31], got %+v", eth1.IPAddresses)
	}
	if eth1.Master != "" {
		t.Errorf("Ethernet1 Master: want '', got %q", eth1.Master)
	}

	pc1, ok := byName["PortChannel1"]
	if !ok {
		t.Fatalf("missing PortChannel1 in result")
	}
	if pc1.AdminStatus != "up" {
		t.Errorf("PortChannel1 admin status: want up, got %s", pc1.AdminStatus)
	}
	if len(pc1.IPAddresses) != 0 {
		t.Errorf("PortChannel1 IPs: want none, got %+v", pc1.IPAddresses)
	}
}

func TestGetInterfacesInNamespace_MasterRelation_And_IPv6(t *testing.T) {
	// Create a master PortChannel100 and a slave Ethernet100 with IPv6.
	linkMsgs := []netlink.Message{
		newRTMNewLinkMsg(10, unix.IFF_UP, "PortChannel100", 0),
		newRTMNewLinkMsg(11, 0, "Ethernet100", 10), // down admin, master=10
	}
	addrMsgs := []netlink.Message{
		newRTMNewAddrMsg(unix.AF_INET6, 11, "2001:db8::1", 64),
	}

	patches := gomonkey.ApplyFunc(unix.Open, func(path string, flags int, mode uint32) (int, error) {
		if flags != unix.O_RDONLY {
			t.Fatalf("unix.Open flags=%d want O_RDONLY", flags)
		}
		if path == "/proc/self/ns/net" {
			return 100, nil
		}
		if strings.HasPrefix(path, "/var/run/netns/") {
			nsname := strings.TrimPrefix(path, "/var/run/netns/")
			if strings.HasPrefix(nsname, "asic") {
				idxStr := strings.TrimPrefix(nsname, "asic")
				idx, err := strconv.Atoi(idxStr)
				if err != nil {
					return -1, fmt.Errorf("invalid namespace index: %s", nsname)
				}
				return 200 + idx, nil
			}
			return -1, fmt.Errorf("unsupported namespace: %s", nsname)
		}
		return -1, fmt.Errorf("unexpected open path: %s", path)
	})
	defer patches.Reset()

	patches.ApplyFunc(unix.Setns, func(fd int, nstype int) error { return nil })
	patches.ApplyFunc(unix.Close, func(fd int) error { return nil })
	patches.ApplyFunc(netlink.Dial, func(family int, cfg *netlink.Config) (*netlink.Conn, error) { return &netlink.Conn{}, nil })
	patches.ApplyMethod(reflect.TypeOf(&netlink.Conn{}), "Close", func(_ *netlink.Conn) error { return nil })
	patches.ApplyMethod(reflect.TypeOf(&netlink.Conn{}), "Execute", func(_ *netlink.Conn, req netlink.Message) ([]netlink.Message, error) {
		switch req.Header.Type {
		case unix.RTM_GETLINK:
			return linkMsgs, nil
		case unix.RTM_GETADDR:
			return addrMsgs, nil
		default:
			return nil, nil
		}
	})

	// Use a non-default namespace to exercise Setns path
	ns := "asic0"
	ifs, err := getInterfacesInNamespace(ns, AddressFamilyIPv6)
	if err != nil {
		t.Fatalf("getInterfacesInNamespace ns=%s failed: %v", ns, err)
	}
	if len(ifs) != 2 {
		// Two links discovered regardless of IP family
		t.Fatalf("expected 2 interfaces, got %d", len(ifs))
	}

	// Build a map by interface name for deterministic assertions
	byName := map[string]IPInterfaceDetail{}
	for _, i := range ifs {
		byName[i.Name] = i
	}

	pc, ok := byName["PortChannel100"]
	if !ok {
		t.Fatalf("missing PortChannel100 in result")
	}
	eth, ok := byName["Ethernet100"]
	if !ok {
		t.Fatalf("missing Ethernet100 in result")
	}

	if pc.AdminStatus != "up" {
		t.Errorf("PortChannel100 admin: want up, got %s", pc.AdminStatus)
	}
	if pc.Master != "" {
		t.Errorf("PortChannel100 master: want '', got %q", pc.Master)
	}
	if eth.AdminStatus != "down" {
		t.Errorf("Ethernet100 admin: want down, got %s", eth.AdminStatus)
	}
	if eth.Master != "PortChannel100" {
		t.Errorf("Ethernet100 master: want PortChannel100, got %q", eth.Master)
	}
	if len(eth.IPAddresses) != 1 || eth.IPAddresses[0].Address != "2001:db8::1/64" {
		t.Errorf("Ethernet100 IPv6: want 2001:db8::1/64, got %+v", eth.IPAddresses)
	}
}

func TestOpenRouteConn_Error(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(netlink.Dial, func(proto int, cfg *netlink.Config) (*netlink.Conn, error) { return nil, errors.New("dial fail") })
	if _, err := openRouteConn(); err == nil {
		t.Fatalf("expected error from openRouteConn when Dial fails")
	}
}

func TestDumpErrors_Propagate(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Stub switchNamespace to no-op restore using existing behavior (default ns)
	patches.ApplyFunc(unix.Open, func(path string, flags int, mode uint32) (int, error) { return 1, nil })
	patches.ApplyFunc(unix.Close, func(fd int) error { return nil })
	patches.ApplyFunc(unix.Setns, func(fd int, nstype int) error { return nil })

	// Conn that fails on link dump
	type fakeConn struct{}
	patches.ApplyFunc(netlink.Dial, func(proto int, cfg *netlink.Config) (*netlink.Conn, error) { return &netlink.Conn{}, nil })
	patches.ApplyMethod(reflect.TypeOf(&netlink.Conn{}), "Close", func(_ *netlink.Conn) error { return nil })
	patches.ApplyMethod(reflect.TypeOf(&netlink.Conn{}), "Execute", func(_ *netlink.Conn, req netlink.Message) ([]netlink.Message, error) {
		if req.Header.Type == unix.RTM_GETLINK {
			return nil, errors.New("dump links fail")
		}
		return nil, nil
	})
	if _, err := getInterfacesInNamespace(defaultNamespace, AddressFamilyIPv4); err == nil {
		t.Fatalf("expected error when dumpLinks fails")
	}

	// Now make links pass and addrs fail
	patches.Reset()
	patches = gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(unix.Open, func(path string, flags int, mode uint32) (int, error) { return 1, nil })
	patches.ApplyFunc(unix.Close, func(fd int) error { return nil })
	patches.ApplyFunc(unix.Setns, func(fd int, nstype int) error { return nil })
	patches.ApplyFunc(netlink.Dial, func(proto int, cfg *netlink.Config) (*netlink.Conn, error) { return &netlink.Conn{}, nil })
	patches.ApplyMethod(reflect.TypeOf(&netlink.Conn{}), "Close", func(_ *netlink.Conn) error { return nil })
	patches.ApplyMethod(reflect.TypeOf(&netlink.Conn{}), "Execute", func(_ *netlink.Conn, req netlink.Message) ([]netlink.Message, error) {
		if req.Header.Type == unix.RTM_GETLINK {
			// Return minimal link with invalid data to exercise parse skip
			b := make([]byte, unix.SizeofIfInfomsg-1) // too short -> parseLinks skip
			return []netlink.Message{{Header: netlink.Header{Type: unix.RTM_NEWLINK}, Data: b}}, nil
		}
		return nil, errors.New("dump addrs fail")
	})
	if _, err := getInterfacesInNamespace(defaultNamespace, AddressFamilyIPv4); err == nil {
		t.Fatalf("expected error when dumpAddrs fails")
	}
}

func TestAssembleInterfaces_FamilySelection_And_AdminFlags(t *testing.T) {
	links := map[int32]linkInfo{
		1: {name: "eth0", masterIndex: 0, flags: unix.IFF_UP},
		2: {name: "eth1", masterIndex: 1, flags: 0},
	}
	v4 := map[int32][]IPAddressDetail{1: {{Address: "192.0.2.1/31"}}}
	v6 := map[int32][]IPAddressDetail{2: {{Address: "2001:db8::2/64"}}}

	got4 := assembleInterfaces(links, v4, v6, AddressFamilyIPv4)
	if len(got4) != 2 {
		t.Fatalf("want 2 entries, got %d", len(got4))
	}
	by := map[string]IPInterfaceDetail{}
	for _, i := range got4 {
		by[i.Name] = i
	}
	if by["eth0"].AdminStatus != "up" || len(by["eth0"].IPAddresses) != 1 {
		t.Fatalf("eth0 unexpected: %+v", by["eth0"])
	}
	if by["eth1"].AdminStatus != "down" || by["eth1"].Master != "eth0" || len(by["eth1"].IPAddresses) != 0 {
		t.Fatalf("eth1 unexpected: %+v", by["eth1"])
	}

	got6 := assembleInterfaces(links, v4, v6, AddressFamilyIPv6)
	by = map[string]IPInterfaceDetail{}
	for _, i := range got6 {
		by[i.Name] = i
	}
	if len(by["eth1"].IPAddresses) != 1 || by["eth1"].IPAddresses[0].Address != "2001:db8::2/64" {
		t.Fatalf("ipv6 selection failed: %+v", by["eth1"])
	}
}

func TestParseAddresses_SupportsIFA_ADDRESS(t *testing.T) {
	// Build RTM_NEWADDR with only IFA_ADDRESS set
	ifa := make([]byte, unix.SizeofIfAddrmsg)
	ifa[0] = unix.AF_INET
	ifa[1] = byte(31)
	copy(ifa[4:8], nlenc.Uint32Bytes(3))
	attrBytes, _ := netlink.MarshalAttributes([]netlink.Attribute{{Type: unix.IFA_ADDRESS, Data: []byte{192, 0, 2, 5}}})
	msgs := []netlink.Message{{Header: netlink.Header{Type: unix.RTM_NEWADDR}, Data: append(ifa, attrBytes...)}}

	v4, v6 := parseAddresses(msgs)
	if len(v4[3]) != 1 || v4[3][0].Address != "192.0.2.5/31" || len(v6) != 0 {
		t.Fatalf("parseAddresses failed: v4=%+v v6=%+v", v4, v6)
	}
}

func TestMarshalHelpers(t *testing.T) {
	b := marshalIfInfomsg(unix.AF_INET6)
	if len(b) != unix.SizeofIfInfomsg || b[0] != unix.AF_INET6 {
		t.Fatalf("marshalIfInfomsg bad: %v", b)
	}
	b = marshalIfAddrmsg(unix.AF_INET)
	if len(b) != unix.SizeofIfAddrmsg || b[0] != unix.AF_INET {
		t.Fatalf("marshalIfAddrmsg bad: %v", b)
	}
}

func TestGetAdminStatusFromFlags(t *testing.T) {
	if getAdminStatusFromFlags(unix.IFF_UP) != "up" {
		t.Fatalf("IFF_UP should be up")
	}
	if getAdminStatusFromFlags(0) != "down" {
		t.Fatalf("0 flags should be down")
	}
}

func TestSwitchNamespace_OpenOriginalFails(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(unix.Open, func(path string, flags int, mode uint32) (int, error) {
		if path == "/proc/self/ns/net" {
			return -1, errors.New("open orig fail")
		}
		return -1, nil
	})
	if _, err := getInterfacesInNamespace("asic0", AddressFamilyIPv4); err == nil {
		t.Fatalf("expected error when original netns open fails")
	}
}

func TestSwitchNamespace_OpenTargetFails(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(unix.Open, func(path string, flags int, mode uint32) (int, error) {
		if path == "/proc/self/ns/net" {
			return 10, nil
		}
		if strings.HasPrefix(path, "/var/run/netns/") {
			return -1, errors.New("open target fail")
		}
		return -1, nil
	})
	patches.ApplyFunc(unix.Close, func(fd int) error { return nil })
	if _, err := getInterfacesInNamespace("asic1", AddressFamilyIPv4); err == nil {
		t.Fatalf("expected error when target netns open fails")
	}
}

func TestSwitchNamespace_SetnsFails(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(unix.Open, func(path string, flags int, mode uint32) (int, error) {
		if path == "/proc/self/ns/net" {
			return 10, nil
		}
		if strings.HasPrefix(path, "/var/run/netns/") {
			return 20, nil
		}
		return -1, nil
	})
	patches.ApplyFunc(unix.Close, func(fd int) error { return nil })
	patches.ApplyFunc(unix.Setns, func(fd int, nstype int) error {
		if fd == 20 && nstype == unix.CLONE_NEWNET {
			return errors.New("setns fail")
		}
		return nil
	})
	if _, err := getInterfacesInNamespace("asic2", AddressFamilyIPv6); err == nil {
		t.Fatalf("expected error when Setns to target fails")
	}
}
