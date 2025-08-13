package ipinterfaces

import (
	"fmt"
	"reflect"
	"testing"
)

func TestGetBGPNeighbors_DefaultNamespace_OK(t *testing.T) {
	var captured [][]string
	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) {
		captured = q
		return map[string]interface{}{
			"BGP_NEIGHBOR|10.0.0.2": map[string]interface{}{"local_addr": "192.0.2.1", "name": "peer1"},
		}, nil
	}
	defer func() { DBQuery = saved }()

	got, err := getBGPNeighborsFromDB("")
	if err != nil {
		t.Fatalf("getBGPNeighborsFromDB error: %v", err)
	}
	wantQuery := [][]string{{"CONFIG_DB", "BGP_NEIGHBOR"}}
	if !reflect.DeepEqual(captured, wantQuery) {
		t.Fatalf("query mismatch: got %v want %v", captured, wantQuery)
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
	var captured [][]string
	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) {
		captured = q
		return map[string]interface{}{
			"BGP_NEIGHBOR|10.0.0.3": map[string]interface{}{"local_addr": "192.0.2.2", "name": "peer2"},
			"BGP_NEIGHBOR|10.0.0.4": map[string]interface{}{"local_addr": "192.0.2.3", "name": "peer3"},
		}, nil
	}
	defer func() { DBQuery = saved }()

	got, err := getBGPNeighborsFromDB("asic1")
	if err != nil {
		t.Fatalf("getBGPNeighborsFromDB error: %v", err)
	}
	wantPrefix := "CONFIG_DB/asic1"
	if len(captured) != 1 || len(captured[0]) != 2 || captured[0][0] != wantPrefix || captured[0][1] != "BGP_NEIGHBOR" {
		t.Fatalf("query mismatch: got %v", captured)
	}
	if len(got) != 2 {
		t.Fatalf("neighbors len: got %d want 2", len(got))
	}
	if got["192.0.2.2"].NeighborIP != "10.0.0.3" || got["192.0.2.3"].NeighborIP != "10.0.0.4" {
		t.Fatalf("neighbors content mismatch: %+v", got)
	}
}

func TestGetBGPNeighbors_SkipInvalidEntries(t *testing.T) {
	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) {
		return map[string]interface{}{
			// Missing local_addr -> should be skipped
			"BGP_NEIGHBOR|10.0.0.5": map[string]interface{}{"name": "peerX"},
			// Not a map -> should be skipped
			"BGP_NEIGHBOR|10.0.0.6": "badtype",
			// Valid entry
			"BGP_NEIGHBOR|10.0.0.7": map[string]interface{}{"local_addr": "192.0.2.7", "name": "peer7"},
		}, nil
	}
	defer func() { DBQuery = saved }()

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
	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) { return nil, fmt.Errorf("db error") }
	defer func() { DBQuery = saved }()

	got, err := getBGPNeighborsFromDB("asic2")
	if err == nil {
		t.Fatalf("expected error, got nil, result=%v", got)
	}
}

func TestGetBGPNeighbors_DBQueryNil_Error(t *testing.T) {
	saved := DBQuery
	DBQuery = nil
	defer func() { DBQuery = saved }()

	if _, err := getBGPNeighborsFromDB(""); err == nil {
		t.Fatalf("expected error when DBQuery is nil")
	}
}

func TestGetBGPNeighbors_NameNonString_Coerced(t *testing.T) {
	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"BGP_NEIGHBOR|203.0.113.2": map[string]interface{}{"local_addr": "192.0.2.9", "name": 12345},
		}, nil
	}
	defer func() { DBQuery = saved }()

	got, err := getBGPNeighborsFromDB("")
	if err != nil {
		t.Fatalf("getBGPNeighborsFromDB error: %v", err)
	}
	n := got["192.0.2.9"]
	if n == nil || n.Name != "12345" || n.NeighborIP != "203.0.113.2" {
		t.Fatalf("unexpected coercion: %+v", n)
	}
}

func TestGetBGPNeighbors_MalformedKey_Skipped(t *testing.T) {
	saved := DBQuery
	DBQuery = func(q [][]string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"BGP_NEIGHBOR": map[string]interface{}{"local_addr": "192.0.2.20", "name": "peer"}, // no delimiter
		}, nil
	}
	defer func() { DBQuery = saved }()

	got, err := getBGPNeighborsFromDB("")
	if err != nil {
		t.Fatalf("getBGPNeighborsFromDB error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no entries due to malformed key, got: %+v", got)
	}
}
