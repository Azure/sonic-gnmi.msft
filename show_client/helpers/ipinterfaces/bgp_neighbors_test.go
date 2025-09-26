package ipinterfaces

import (
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

func TestGetBGPNeighbors_DefaultNamespace_OK(t *testing.T) {
	p := gomonkey.ApplyFunc(common.GetMapFromQueries, func(q [][]string) (map[string]interface{}, error) {
		// Expect exactly [["CONFIG_DB", "BGP_NEIGHBOR"]]
		if len(q) == 1 && len(q[0]) == 2 && q[0][0] == "CONFIG_DB" && q[0][1] == "BGP_NEIGHBOR" {
			return map[string]interface{}{
				"10.0.0.2": map[string]interface{}{"local_addr": "192.0.2.1", "name": "peer1"},
			}, nil
		}
		return nil, fmt.Errorf("unexpected query: %v", q)
	})
	defer p.Reset()

	got, err := getBGPNeighborsFromDB("")
	if err != nil {
		t.Fatalf("getBGPNeighborsFromDB error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("neighbors len: got %d want 1", len(got))
	}
	n, ok := got["192.0.2.1"]
	if !ok {
		t.Fatalf("missing key 192.0.2.1")
	}
	if n.NeighborIP != "10.0.0.2" || n.Name != "peer1" {
		t.Fatalf("neighbor mismatch: got %+v", n)
	}
}

func TestGetBGPNeighbors_NonDefaultNamespace_OK(t *testing.T) {
	p := gomonkey.ApplyFunc(common.GetMapFromQueries, func(q [][]string) (map[string]interface{}, error) {
		// Expect exactly [["CONFIG_DB/asic1", "BGP_NEIGHBOR"]]
		if len(q) == 1 && len(q[0]) == 2 && q[0][0] == "CONFIG_DB/asic1" && q[0][1] == "BGP_NEIGHBOR" {
			return map[string]interface{}{
				"10.0.0.3": map[string]interface{}{"local_addr": "192.0.2.2", "name": "peer2"},
				"10.0.0.4": map[string]interface{}{"local_addr": "192.0.2.3", "name": "peer3"},
			}, nil
		}
		return nil, fmt.Errorf("unexpected query: %v", q)
	})
	defer p.Reset()

	got, err := getBGPNeighborsFromDB("asic1")
	if err != nil {
		t.Fatalf("getBGPNeighborsFromDB error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("neighbors len: got %d want 2", len(got))
	}
	if got["192.0.2.2"].NeighborIP != "10.0.0.3" || got["192.0.2.3"].NeighborIP != "10.0.0.4" {
		t.Fatalf("neighbors content mismatch: %+v", got)
	}
}

func TestGetBGPNeighbors_SkipInvalidEntries(t *testing.T) {
	p := gomonkey.ApplyFunc(common.GetMapFromQueries, func(q [][]string) (map[string]interface{}, error) {
		return map[string]interface{}{
			// Missing local_addr -> should be skipped
			"10.0.0.5": map[string]interface{}{"name": "peerX"},
			// Not a map -> should be skipped
			"10.0.0.6": "badtype",
			// Valid entry
			"10.0.0.7": map[string]interface{}{"local_addr": "192.0.2.7", "name": "peer7"},
		}, nil
	})
	defer p.Reset()

	got, err := getBGPNeighborsFromDB("")
	if err != nil {
		t.Fatalf("getBGPNeighborsFromDB error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("neighbors len: got %d want 1", len(got))
	}
	n := got["192.0.2.7"]
	if n == nil || n.NeighborIP != "10.0.0.7" || n.Name != "peer7" {
		t.Fatalf("neighbor mismatch: %+v", n)
	}
}

func TestGetBGPNeighbors_Error(t *testing.T) {
	p := gomonkey.ApplyFunc(common.GetMapFromQueries, func(q [][]string) (map[string]interface{}, error) {
		return nil, fmt.Errorf("db error")
	})
	defer p.Reset()

	got, err := getBGPNeighborsFromDB("asic2")
	if err == nil {
		t.Fatalf("expected error, got nil, result=%v", got)
	}
}

func TestGetBGPNeighbors_NameNonString_Coerced(t *testing.T) {
	p := gomonkey.ApplyFunc(common.GetMapFromQueries, func(q [][]string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"203.0.113.2": map[string]interface{}{"local_addr": "192.0.2.9", "name": 12345},
		}, nil
	})
	defer p.Reset()

	got, err := getBGPNeighborsFromDB("")
	if err != nil {
		t.Fatalf("getBGPNeighborsFromDB error: %v", err)
	}
	n := got["192.0.2.9"]
	// db.go only accepts string 'name' values; non-string names are ignored and default to empty
	if n == nil || n.Name != "" || n.NeighborIP != "203.0.113.2" {
		t.Fatalf("unexpected result for non-string name: %+v", n)
	}
}

func TestGetBGPNeighbors_MalformedKey_Skipped(t *testing.T) {
	p := gomonkey.ApplyFunc(common.GetMapFromQueries, func(q [][]string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"BGP_NEIGHBOR": map[string]interface{}{"local_addr": "192.0.2.20", "name": "peer"}, // not an IP -> skipped
		}, nil
	})
	defer p.Reset()

	got, err := getBGPNeighborsFromDB("")
	if err != nil {
		t.Fatalf("getBGPNeighborsFromDB error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no entries due to malformed key, got: %+v", got)
	}
}
