package ipinterfaces

import (
	"bytes"
	"fmt"
	"net"
	"runtime"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"golang.org/x/sys/unix"
)

// getInterfacesInNamespace retrieves all interface details from a given network namespace.
func getInterfacesInNamespace(namespace string, addressFamily string) ([]IPInterfaceDetail, error) {
	// Ensure namespace ops happen on a single OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Switch to target namespace if needed and ensure we restore on exit.
	restore, err := switchNamespace(namespace)
	if err != nil {
		return nil, err
	}
	defer restore()

	// Open rtnetlink (NETLINK_ROUTE).
	conn, err := openRouteConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Dump links and addresses.
	linkMsgs, err := dumpLinks(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to list links in namespace '%s': %w", namespace, err)
	}
	addrMsgs, err := dumpAddrs(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to list addresses in namespace '%s': %w", namespace, err)
	}

	// Parse.
	links := parseLinks(linkMsgs)
	v4ByIdx, v6ByIdx := parseAddresses(addrMsgs)

	// Assemble.
	interfaces := assembleInterfaces(links, v4ByIdx, v6ByIdx, addressFamily)
	return interfaces, nil
}

// switchNamespace switches into the requested namespace if it is not the default namespace
// and returns a restore function to switch back. If namespace is the default, it returns a no-op restore.
func switchNamespace(namespace string) (func(), error) {
	// Always capture the original namespace FD so we can restore later.
	origFD, err := unix.Open("/proc/self/ns/net", unix.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open original netns: %w", err)
	}

	// Default namespace: no switch, just provide a restore that closes origFD.
	if namespace == defaultNamespace {
		return func() { _ = unix.Close(origFD) }, nil
	}

	// Open target namespace and switch.
	targetPath := "/var/run/netns/" + namespace
	targetFD, err := unix.Open(targetPath, unix.O_RDONLY, 0)
	if err != nil {
		_ = unix.Close(origFD)
		return nil, fmt.Errorf("failed to open target namespace '%s': %w", namespace, err)
	}
	if err := unix.Setns(targetFD, unix.CLONE_NEWNET); err != nil {
		_ = unix.Close(targetFD)
		_ = unix.Close(origFD)
		return nil, fmt.Errorf("failed to switch to namespace '%s': %w", namespace, err)
	}

	// Restore closure switches back and closes FDs.
	return func() {
		_ = unix.Setns(origFD, unix.CLONE_NEWNET)
		_ = unix.Close(targetFD)
		_ = unix.Close(origFD)
	}, nil
}

// openRouteConn dials a NETLINK_ROUTE socket.
func openRouteConn() (*netlink.Conn, error) {
	conn, err := netlink.Dial(unix.NETLINK_ROUTE, &netlink.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open netlink route socket: %w", err)
	}
	return conn, nil
}

// dumpLinks requests RTM_GETLINK dump.
func dumpLinks(conn *netlink.Conn) ([]netlink.Message, error) {
	return conn.Execute(netlink.Message{
		Header: netlink.Header{Type: unix.RTM_GETLINK, Flags: netlink.Request | netlink.Dump},
		Data:   marshalIfInfomsg(unix.AF_UNSPEC),
	})
}

// dumpAddrs requests RTM_GETADDR dump.
func dumpAddrs(conn *netlink.Conn) ([]netlink.Message, error) {
	return conn.Execute(netlink.Message{
		Header: netlink.Header{Type: unix.RTM_GETADDR, Flags: netlink.Request | netlink.Dump},
		Data:   marshalIfAddrmsg(unix.AF_UNSPEC),
	})
}

// linkInfo captures minimal link attributes we need.
type linkInfo struct {
	name        string
	masterIndex int32
	flags       uint32
	carrier     *bool // nil = not present, non-nil = present (true=up, false=down)
}

// parseLinks builds linkInfo keyed by ifindex from RTM_NEWLINK messages.
func parseLinks(linkMsgs []netlink.Message) map[int32]linkInfo {
	links := make(map[int32]linkInfo)
	for _, m := range linkMsgs {
		if len(m.Data) < unix.SizeofIfInfomsg {
			continue
		}
		// struct ifinfomsg layout: family(0), pad(1), type(2-3), index(4-7), flags(8-11), change(12-15)
		idx := int32(nlenc.Uint32(m.Data[4:8]))
		flags := nlenc.Uint32(m.Data[8:12])
		attrs, err := netlink.UnmarshalAttributes(m.Data[unix.SizeofIfInfomsg:])
		if err != nil {
			continue
		}
		li := linkInfo{}
		for _, a := range attrs {
			switch a.Type {
			case unix.IFLA_IFNAME:
				li.name = string(bytes.TrimRight(a.Data, "\x00"))
			case unix.IFLA_MASTER:
				li.masterIndex = int32(nlenc.Uint32(a.Data))
			case unix.IFLA_CARRIER:
				if len(a.Data) > 0 {
					up := a.Data[0] != 0
					li.carrier = &up
				}
			}
		}
		if li.name == "" || idx == 0 {
			continue
		}
		li.flags = flags
		links[idx] = li
	}
	return links
}

// parseAddresses builds per-ifindex address maps for IPv4 and IPv6 from RTM_NEWADDR messages.
func parseAddresses(addrMsgs []netlink.Message) (map[int32][]IPAddressDetail, map[int32][]IPAddressDetail) {
	v4 := make(map[int32][]IPAddressDetail)
	v6 := make(map[int32][]IPAddressDetail)
	for _, m := range addrMsgs {
		if len(m.Data) < unix.SizeofIfAddrmsg {
			continue
		}
		family := m.Data[0]
		prefix := int(m.Data[1])
		idx := int32(nlenc.Uint32(m.Data[4:8])) // ifa_index
		attrs, err := netlink.UnmarshalAttributes(m.Data[unix.SizeofIfAddrmsg:])
		if err != nil {
			continue
		}
		var ip net.IP
		for _, a := range attrs {
			switch a.Type {
			case unix.IFA_LOCAL:
				ip = net.IP(a.Data)
			case unix.IFA_ADDRESS:
				if ip == nil {
					ip = net.IP(a.Data)
				}
			}
		}
		if ip == nil {
			continue
		}
		ipd := IPAddressDetail{Address: fmt.Sprintf("%s/%d", ip.String(), prefix)}
		if family == unix.AF_INET {
			v4[idx] = append(v4[idx], ipd)
		} else if family == unix.AF_INET6 {
			v6[idx] = append(v6[idx], ipd)
		}
	}
	return v4, v6
}

// assembleInterfaces merges link and address data into the final []IPInterfaceDetail.
func assembleInterfaces(links map[int32]linkInfo, v4, v6 map[int32][]IPAddressDetail, family string) []IPInterfaceDetail {
	var interfaces []IPInterfaceDetail
	for ifidx, li := range links {
		var ipAddresses []IPAddressDetail
		if family == AddressFamilyIPv4 {
			ipAddresses = v4[ifidx]
		} else if family == AddressFamilyIPv6 {
			ipAddresses = v6[ifidx]
		}
		// Skip pure L2 interfaces (no addresses for the selected family) to mirror python ipintutil behavior.
		if len(ipAddresses) == 0 {
			continue
		}
		masterName := ""
		if li.masterIndex != 0 {
			if mli, ok := links[li.masterIndex]; ok {
				masterName = mli.name
			}
		}
		interfaces = append(interfaces, IPInterfaceDetail{
			Name:        li.name,
			IPAddresses: ipAddresses,
			AdminStatus: getAdminStatus(li),
			OperStatus:  getOperStatus(li),
			Master:      masterName,
		})
	}
	return interfaces
}

// marshalIfInfomsg returns a minimal ifinfomsg payload for RTM_GETLINK.
func marshalIfInfomsg(family uint8) []byte {
	b := make([]byte, unix.SizeofIfInfomsg)
	b[0] = family // ifi_family
	return b
}

// marshalIfAddrmsg returns a minimal ifaddrmsg payload for RTM_GETADDR.
func marshalIfAddrmsg(family uint8) []byte {
	b := make([]byte, unix.SizeofIfAddrmsg)
	b[0] = family // ifa_family
	return b
}

// getAdminStatus translates linkInfo to an admin status string ("up" or "down").
func getAdminStatus(li linkInfo) string {
	if li.flags&unix.IFF_UP != 0 {
		return "up"
	}
	return "down"
}

// getOperStatus translates linkInfo to an oper status string ("up" or "down").
// - Prefer IFLA_CARRIER when available (carrier != nil).
// - Else use IFF_LOWER_UP (flags) as the carrier indicator.
func getOperStatus(li linkInfo) string {
	if li.carrier != nil {
		if *li.carrier {
			return "up"
		}
		return "down"
	}
	if li.flags&unix.IFF_LOWER_UP != 0 {
		return "up"
	}
	return "down"
}
