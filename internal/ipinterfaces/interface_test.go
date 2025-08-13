package ipinterfaces

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
)

func TestGetIPInterfaces_SingleASIC_WithBGPEnrichment(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Single ASIC path
	patches.ApplyFunc(IsMultiASIC, func() (bool, error) { return false, nil })

	// Stub interface data from default namespace
	patches.ApplyFunc(getInterfacesInNamespace, func(ns, af string) ([]IPInterfaceDetail, error) {
		if ns != "" {
			t.Fatalf("expected default namespace, got %q", ns)
		}
		return []IPInterfaceDetail{{
			Name:        "Ethernet0",
			IPAddresses: []IPAddressDetail{{Address: "192.0.2.1/31"}},
			AdminStatus: "up",
			OperStatus:  "up",
		}}, nil
	})

	// Inject BGP data via DBQuery
	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"BGP_NEIGHBOR|10.0.0.2": map[string]interface{}{"local_addr": "192.0.2.1", "name": "peerA"},
		}, nil
	}
	defer func() { DBQuery = saved }()

	got, err := GetIPInterfaces(AddressFamilyIPv4, nil, nil)
	if err != nil {
		t.Fatalf("GetIPInterfaces error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Ethernet0" {
		t.Fatalf("unexpected interfaces: %+v", got)
	}
	if len(got[0].IPAddresses) != 1 {
		t.Fatalf("unexpected IPs: %+v", got[0].IPAddresses)
	}
	ip := got[0].IPAddresses[0]
	if ip.BGPNeighborIP != "10.0.0.2" || ip.BGPNeighborName != "peerA" {
		t.Fatalf("BGP enrichment failed: %+v", ip)
	}
}

func TestGetIPInterfaces_MultiASIC_MergesAndAppendsDefault(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Multi-ASIC and one frontend namespace
	patches.ApplyFunc(IsMultiASIC, func() (bool, error) { return true, nil })
	patches.ApplyFunc(GetAllNamespaces, func() (*NamespacesByRole, error) {
		return &NamespacesByRole{Frontend: []string{"asic0"}}, nil
	})

	// Return different IPs for same interface name across namespaces
	patches.ApplyFunc(getInterfacesInNamespace, func(ns, af string) ([]IPInterfaceDetail, error) {
		switch ns {
		case "asic0":
			return []IPInterfaceDetail{{
				Name:        "Ethernet0",
				IPAddresses: []IPAddressDetail{{Address: "2001:db8::1/64"}},
			}}, nil
		case "":
			return []IPInterfaceDetail{{
				Name:        "Ethernet0",
				IPAddresses: []IPAddressDetail{{Address: "192.0.2.2/31"}},
			}}, nil
		default:
			t.Fatalf("unexpected namespace %q", ns)
			return nil, nil
		}
	})

	// Inject empty BGP to focus on merging behavior
	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) { return map[string]interface{}{}, nil }
	defer func() { DBQuery = saved }()

	got, err := GetIPInterfaces(AddressFamilyIPv6, nil, nil)
	if err != nil {
		t.Fatalf("GetIPInterfaces error: %v", err)
	}
	// Expect one interface Ethernet0 with both IPs merged (order not guaranteed)
	if len(got) != 1 || got[0].Name != "Ethernet0" {
		t.Fatalf("unexpected interfaces: %+v", got)
	}
	ips := got[0].IPAddresses
	if len(ips) != 2 {
		t.Fatalf("expected 2 IPs merged, got %+v", ips)
	}
	set := map[string]bool{}
	for _, ip := range ips {
		set[ip.Address] = true
	}
	expected := map[string]bool{"2001:db8::1/64": true, "192.0.2.2/31": true}
	if !reflect.DeepEqual(set, expected) {
		t.Fatalf("merged IPs mismatch: got %v want %v", set, expected)
	}
}

func TestGetIPInterfaces_UnsupportedFamily_ReturnsError(t *testing.T) {
	if _, err := GetIPInterfaces("ipv5", nil, nil); err == nil {
		t.Fatalf("expected error for unsupported address family")
	}
}

func TestGetIPInterfaces_SingleASIC_UnknownNamespace_Error(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(IsMultiASIC, func() (bool, error) { return false, nil })
	// Ensure getInterfacesInNamespace is NOT called
	patches.ApplyFunc(getInterfacesInNamespace, func(ns, af string) ([]IPInterfaceDetail, error) {
		t.Fatalf("getInterfacesInNamespace should not be called on invalid namespace in single-ASIC")
		return nil, nil
	})

	ns := "asic0"
	if _, err := GetIPInterfaces(AddressFamilyIPv4, &ns, nil); err == nil {
		t.Fatalf("expected error for unknown namespace in single-ASIC mode")
	}
}

func TestGetIPInterfaces_MultiASIC_ExplicitNamespace_Dedup(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(IsMultiASIC, func() (bool, error) { return true, nil })
	patches.ApplyFunc(GetAllNamespaces, func() (*NamespacesByRole, error) {
		return &NamespacesByRole{Frontend: []string{"asic0", "asic1"}, Backend: []string{"asic2"}}, nil
	})

	patches.ApplyFunc(getInterfacesInNamespace, func(ns, af string) ([]IPInterfaceDetail, error) {
		switch ns {
		case "asic2":
			return []IPInterfaceDetail{{
				Name:        "Ethernet0",
				IPAddresses: []IPAddressDetail{{Address: "192.0.2.10/31"}},
			}}, nil
		case "": // default appended
			return []IPInterfaceDetail{{
				Name:        "Ethernet0",
				IPAddresses: []IPAddressDetail{{Address: "192.0.2.10/31"}}, // duplicate to test dedup
			}}, nil
		default:
			return []IPInterfaceDetail{}, nil
		}
	})

	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) { return map[string]interface{}{}, nil }
	defer func() { DBQuery = saved }()

	ns := "asic2"
	frontend := DisplayExternal
	got, err := GetIPInterfaces(AddressFamilyIPv4, &ns, &frontend)
	if err != nil {
		t.Fatalf("GetIPInterfaces error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Ethernet0" {
		t.Fatalf("unexpected interfaces: %+v", got)
	}
	if len(got[0].IPAddresses) != 1 { // duplicate should be removed
		t.Fatalf("expected deduped IPs, got %+v", got[0].IPAddresses)
	}
}

func TestGetIPInterfaces_MultiASIC_FrontendOnly_DefaultNsError_DBQueryNil(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(IsMultiASIC, func() (bool, error) { return true, nil })
	patches.ApplyFunc(GetAllNamespaces, func() (*NamespacesByRole, error) {
		return &NamespacesByRole{Frontend: []string{"asic0", "asic1"}}, nil
	})
	patches.ApplyFunc(getInterfacesInNamespace, func(ns, af string) ([]IPInterfaceDetail, error) {
		switch ns {
		case "asic0":
			return []IPInterfaceDetail{{Name: "Ethernet0", IPAddresses: []IPAddressDetail{{Address: "2001:db8::1/64"}}}}, nil
		case "asic1":
			return nil, nil
		case "":
			return nil, fmt.Errorf("boom") // simulate default namespace failure
		default:
			return nil, nil
		}
	})

	// Force DB enrichment to take error path (DBQuery unset -> getBGPNeighborsFromDB returns error)
	saved := DBQuery
	DBQuery = nil
	defer func() { DBQuery = saved }()

	frontend := DisplayExternal
	got, err := GetIPInterfaces(AddressFamilyIPv6, nil, &frontend)
	if err != nil {
		t.Fatalf("GetIPInterfaces error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Ethernet0" {
		t.Fatalf("unexpected interfaces: %+v", got)
	}
	// No BGP fields set due to enrichment error path
	if len(got[0].IPAddresses) != 1 || got[0].IPAddresses[0].BGPNeighborIP != "" || got[0].IPAddresses[0].BGPNeighborName != "" {
		t.Fatalf("unexpected BGP enrichment: %+v", got[0].IPAddresses)
	}
}

func TestEnrichWithBGPData_InvalidCIDR_Skipped(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Single ASIC to keep it simple
	patches.ApplyFunc(IsMultiASIC, func() (bool, error) { return false, nil })
	patches.ApplyFunc(getInterfacesInNamespace, func(ns, af string) ([]IPInterfaceDetail, error) {
		return []IPInterfaceDetail{{
			Name:        "Ethernet0",
			IPAddresses: []IPAddressDetail{{Address: "not-a-cidr"}},
		}}, nil
	})

	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) {
		return map[string]interface{}{"BGP_NEIGHBOR|10.0.0.2": map[string]interface{}{"local_addr": "203.0.113.1", "name": "peerA"}}, nil
	}
	defer func() { DBQuery = saved }()

	got, err := GetIPInterfaces(AddressFamilyIPv4, nil, nil)
	if err != nil {
		t.Fatalf("GetIPInterfaces error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Ethernet0" {
		t.Fatalf("unexpected interfaces: %+v", got)
	}
	// Invalid CIDR should not set BGP fields
	if got[0].IPAddresses[0].BGPNeighborIP != "" || got[0].IPAddresses[0].BGPNeighborName != "" {
		t.Fatalf("BGP should be skipped for invalid CIDR: %+v", got[0].IPAddresses[0])
	}
}

func TestGetIPInterfaces_MultiASIC_DefaultAlreadyIncluded_NoDuplicateAppend(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(IsMultiASIC, func() (bool, error) { return true, nil })
	patches.ApplyFunc(GetAllNamespaces, func() (*NamespacesByRole, error) {
		// Frontend already includes default namespace
		return &NamespacesByRole{Frontend: []string{"", "asic0"}}, nil
	})
	calls := map[string]int{}
	patches.ApplyFunc(getInterfacesInNamespace, func(ns, af string) ([]IPInterfaceDetail, error) {
		calls[ns]++
		switch ns {
		case "":
			return []IPInterfaceDetail{{Name: "Eth0", IPAddresses: []IPAddressDetail{{Address: "192.0.2.3/31"}}}}, nil
		case "asic0":
			return []IPInterfaceDetail{{Name: "Eth0", IPAddresses: []IPAddressDetail{{Address: "2001:db8::3/64"}}}}, nil
		default:
			return nil, nil
		}
	})
	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) { return map[string]interface{}{}, nil }
	defer func() { DBQuery = saved }()

	got, err := GetIPInterfaces(AddressFamilyIPv4, nil, nil)
	if err != nil {
		t.Fatalf("GetIPInterfaces error: %v", err)
	}
	if calls[""] != 1 {
		t.Fatalf("default namespace should be queried once, got %d", calls[""])
	}
	if calls["asic0"] != 1 {
		t.Fatalf("asic0 should be queried once, got %d", calls["asic0"])
	}
	if len(got) != 1 || got[0].Name != "Eth0" || len(got[0].IPAddresses) != 2 {
		t.Fatalf("unexpected merge result: %+v", got)
	}
}

func TestGetIPInterfaces_MultiASIC_GetAllNamespaces_Error(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(IsMultiASIC, func() (bool, error) { return true, nil })
	patches.ApplyFunc(GetAllNamespaces, func() (*NamespacesByRole, error) { return nil, fmt.Errorf("ns err") })

	if _, err := GetIPInterfaces(AddressFamilyIPv4, nil, nil); err == nil {
		t.Fatalf("expected error when GetAllNamespaces fails")
	}
}

func TestGetIPInterfaces_MultiASIC_ExplicitUnknownNamespace_Error(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(IsMultiASIC, func() (bool, error) { return true, nil })
	patches.ApplyFunc(GetAllNamespaces, func() (*NamespacesByRole, error) {
		return &NamespacesByRole{Frontend: []string{"asic0"}, Backend: []string{"asic1"}}, nil
	})
	ns := "weird"
	if _, err := GetIPInterfaces(AddressFamilyIPv4, &ns, nil); err == nil {
		t.Fatalf("expected error for unknown namespace in multi-ASIC")
	}
}

func TestGetIPInterfaces_MultiASIC_NonDefaultNamespaceError_Ignored(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(IsMultiASIC, func() (bool, error) { return true, nil })
	patches.ApplyFunc(GetAllNamespaces, func() (*NamespacesByRole, error) {
		return &NamespacesByRole{Frontend: []string{"asic0", "asic1"}}, nil
	})
	patches.ApplyFunc(getInterfacesInNamespace, func(ns, af string) ([]IPInterfaceDetail, error) {
		switch ns {
		case "asic0":
			return nil, fmt.Errorf("boom")
		case "asic1":
			return []IPInterfaceDetail{{Name: "Ethernet8", IPAddresses: []IPAddressDetail{{Address: "198.51.100.1/31"}}}}, nil
		case "":
			return []IPInterfaceDetail{{Name: "Ethernet8", IPAddresses: []IPAddressDetail{{Address: "2001:db8::8/64"}}}}, nil
		default:
			return nil, nil
		}
	})
	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) { return map[string]interface{}{}, nil }
	defer func() { DBQuery = saved }()

	got, err := GetIPInterfaces(AddressFamilyIPv6, nil, nil)
	if err != nil {
		t.Fatalf("GetIPInterfaces error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Ethernet8" || len(got[0].IPAddresses) != 2 {
		t.Fatalf("unexpected result when one ns errors: %+v", got)
	}
}
