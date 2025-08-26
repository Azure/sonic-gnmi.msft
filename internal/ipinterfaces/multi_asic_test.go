package ipinterfaces

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
)

func TestGetPlatform_FromEnv(t *testing.T) {
	patches := gomonkey.ApplyFunc(os.Getenv, func(key string) string {
		if key == "PLATFORM" {
			return "x86_64-mlnx_msn2700-r0"
		}
		return ""
	})
	defer patches.Reset()

	// If code still tries to open machine.conf, fail fast
	patches.ApplyFunc(os.Open, func(name string) (*os.File, error) {
		return nil, errors.New("os.Open should not be called when PLATFORM is set")
	})

	got := getPlatform(nil)
	if got != "x86_64-mlnx_msn2700-r0" {
		t.Fatalf("getPlatform from env: got %q, want %q", got, "x86_64-mlnx_msn2700-r0")
	}
}

func TestGetPlatform_FromConfigDB(t *testing.T) {
	dbq := func(q [][]string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"DEVICE_METADATA|localhost": map[string]interface{}{"platform": "arm64-dummy"},
		}, nil
	}
	patches := gomonkey.ApplyFunc(os.Getenv, func(key string) string { return "" })
	defer patches.Reset()

	// Make machine.conf open fail so code falls back to DB, avoid calling os.Open from patch
	patches.ApplyFunc(os.Open, func(name string) (*os.File, error) {
		if name == machineConfPath {
			return nil, os.ErrNotExist
		}
		return nil, os.ErrNotExist
	})

	got := getPlatform(dbq)
	if got != "arm64-dummy" {
		t.Fatalf("getPlatform from DB: got %q, want %q", got, "arm64-dummy")
	}
}

func TestGetNumASICs_NoFile(t *testing.T) {
	// getAsicConfFilePath should return "" causing GetNumASICs to return 1
	patches := gomonkey.ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	})
	defer patches.Reset()

	// Ensure platform lookup doesn’t accidentally find something
	patches.ApplyFunc(os.Getenv, func(key string) string { return "" })
	patches.ApplyFunc(os.Open, func(name string) (*os.File, error) { return nil, os.ErrNotExist })

	n, err := GetNumASICs(nil)
	if err != nil {
		t.Fatalf("GetNumASICs error: %v", err)
	}
	if n != 1 {
		t.Fatalf("GetNumASICs: got %d, want 1", n)
	}
}

func TestGetNumASICs_FromAsicConf(t *testing.T) {
	// Create a temp asic.conf and have getAsicConfFilePath find it as candidate1
	tmpDir := t.TempDir()
	asicConf := filepath.Join(tmpDir, asicConfFilename)
	if err := os.WriteFile(asicConf, []byte("num_asic=4\n"), 0644); err != nil {
		t.Fatal(err)
	}
	candidate1 := filepath.Join(containerPlatformPath, asicConfFilename)

	patches := gomonkey.ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
		if name == candidate1 {
			return fakeFileInfo{name: candidate1}, nil
		}
		return nil, os.ErrNotExist
	})
	defer patches.Reset()

	// Pre-open the temp file; return this handle when candidate1 is opened
	fh, err := os.Open(asicConf)
	if err != nil {
		t.Fatal(err)
	}
	patches.ApplyFunc(os.Open, func(name string) (*os.File, error) {
		if name == candidate1 {
			return fh, nil
		}
		return nil, os.ErrNotExist
	})

	n, err := GetNumASICs(nil)
	if err != nil {
		t.Fatalf("GetNumASICs error: %v", err)
	}
	if n != 4 {
		t.Fatalf("GetNumASICs: got %d, want 4", n)
	}
}

func TestGetNumASICs_MalformedValue_Error(t *testing.T) {
	tmpDir := t.TempDir()
	asicConf := filepath.Join(tmpDir, asicConfFilename)
	if err := os.WriteFile(asicConf, []byte("num_asic=notanint\n"), 0644); err != nil {
		t.Fatal(err)
	}
	candidate1 := filepath.Join(containerPlatformPath, asicConfFilename)

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
		if name == candidate1 {
			return fakeFileInfo{name: candidate1}, nil
		}
		return nil, os.ErrNotExist
	})
	fh, _ := os.Open(asicConf)
	patches.ApplyFunc(os.Open, func(name string) (*os.File, error) {
		if name == candidate1 {
			return fh, nil
		}
		return nil, os.ErrNotExist
	})
	if _, err := GetNumASICs(nil); err == nil {
		t.Fatalf("expected error for malformed num_asic value")
	}
}

func TestGetNumASICs_ScannerError(t *testing.T) {
	// Create a candidate1 asic.conf with a line longer than bufio.MaxScanTokenSize to force Scanner.Err()
	tmpDir := t.TempDir()
	asicConf := filepath.Join(tmpDir, asicConfFilename)
	longLine := strings.Repeat("A", bufio.MaxScanTokenSize+1024) + "\n"
	if err := os.WriteFile(asicConf, []byte(longLine), 0644); err != nil {
		t.Fatal(err)
	}
	candidate1 := filepath.Join(containerPlatformPath, asicConfFilename)

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
		if name == candidate1 {
			return fakeFileInfo{name: candidate1}, nil
		}
		return nil, os.ErrNotExist
	})
	fh, err := os.Open(asicConf)
	if err != nil {
		t.Fatal(err)
	}
	patches.ApplyFunc(os.Open, func(name string) (*os.File, error) {
		if name == candidate1 {
			return fh, nil
		}
		return nil, os.ErrNotExist
	})

	if _, err := GetNumASICs(nil); err == nil {
		t.Fatalf("expected scanner error due to oversized line")
	}
}

func TestIsMultiASIC_ErrorPropagates(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(GetNumASICs, func(DBQueryFunc) (int, error) { return 0, fmt.Errorf("boom") })
	if _, err := IsMultiASIC(nil); err == nil {
		t.Fatalf("expected error to propagate")
	}
}

func TestGetAllNamespaces_SingleASIC(t *testing.T) {
	patches := gomonkey.ApplyFunc(GetNumASICs, func(DBQueryFunc) (int, error) { return 1, nil })
	defer patches.Reset()

	ns, err := GetAllNamespaces(DiscardLogger, nil)
	if err != nil {
		t.Fatalf("GetAllNamespaces error: %v", err)
	}
	want := &NamespacesByRole{Frontend: []string{defaultNamespace}}
	if !reflect.DeepEqual(ns, want) {
		t.Fatalf("GetAllNamespaces single: got %+v, want %+v", ns, want)
	}
}

func TestGetAllNamespaces_MultiASIC(t *testing.T) {
	dbq := func(q [][]string) (map[string]interface{}, error) {
		if len(q) == 0 || len(q[0]) == 0 {
			return nil, fmt.Errorf("bad query")
		}
		db := q[0][0] // e.g., CONFIG_DB/asic0
		role := ""
		if strings.Contains(db, "asic0") {
			role = "Frontend"
		} else if strings.Contains(db, "asic1") {
			role = "Backend"
		}
		return map[string]interface{}{
			"DEVICE_METADATA|localhost": map[string]interface{}{"sub_role": role},
		}, nil
	}
	patches := gomonkey.ApplyFunc(GetNumASICs, func(DBQueryFunc) (int, error) { return 2, nil })
	defer patches.Reset()

	ns, err := GetAllNamespaces(DiscardLogger, dbq)
	if err != nil {
		t.Fatalf("GetAllNamespaces error: %v", err)
	}
	// Order isn’t guaranteed; compare via sets
	gotF := append([]string{}, ns.Frontend...)
	gotB := append([]string{}, ns.Backend...)
	if !sameStringSet(gotF, []string{"asic0"}) || !sameStringSet(gotB, []string{"asic1"}) {
		t.Fatalf("GetAllNamespaces multi: got F=%v B=%v", gotF, gotB)
	}
}

func TestGetAllNamespaces_DBFailureAndFabricRole(t *testing.T) {
	calls := 0
	dbq := func(q [][]string) (map[string]interface{}, error) {
		calls++
		db := q[0][0]
		if strings.Contains(db, "asic0") {
			return nil, fmt.Errorf("db fail") // should be skipped with warning
		}
		role := "Backend"
		if strings.Contains(db, "asic2") {
			role = "Fabric"
		}
		return map[string]interface{}{"DEVICE_METADATA|localhost": map[string]interface{}{"sub_role": role}}, nil
	}
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(GetNumASICs, func(DBQueryFunc) (int, error) { return 3, nil })

	ns, err := GetAllNamespaces(DiscardLogger, dbq)
	if err != nil {
		t.Fatalf("GetAllNamespaces error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 DB calls, got %d", calls)
	}
	if !sameStringSet(ns.Backend, []string{"asic1"}) || !sameStringSet(ns.Fabric, []string{"asic2"}) {
		t.Fatalf("unexpected namespaces: %+v", ns)
	}
}

func TestGetAllNamespaces_DBQueryNil_MultiASIC(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	// Force multi-ASIC path (3 ASICs) but provide nil DBQuery so all role detections are skipped
	patches.ApplyFunc(GetNumASICs, func(DBQueryFunc) (int, error) { return 3, nil })
	ns, err := GetAllNamespaces(DiscardLogger, nil)
	if err != nil {
		t.Fatalf("GetAllNamespaces error: %v", err)
	}
	if len(ns.Frontend) != 0 || len(ns.Backend) != 0 || len(ns.Fabric) != 0 {
		t.Fatalf("expected all role slices empty with nil DBQuery, got %+v", ns)
	}
}

func TestGetAllNamespaces_RoleClassification_UnknownAndError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	// 5 ASICs: front/back/fabric + unknown role + error
	patches.ApplyFunc(GetNumASICs, func(DBQueryFunc) (int, error) { return 5, nil })
	calls := 0
	dbq := func(q [][]string) (map[string]interface{}, error) {
		calls++
		src := q[0][0]
		switch src {
		case "CONFIG_DB/asic0":
			return map[string]interface{}{"DEVICE_METADATA|localhost": map[string]interface{}{"sub_role": "Frontend"}}, nil
		case "CONFIG_DB/asic1":
			return map[string]interface{}{"DEVICE_METADATA|localhost": map[string]interface{}{"sub_role": "Backend"}}, nil
		case "CONFIG_DB/asic2":
			return map[string]interface{}{"DEVICE_METADATA|localhost": map[string]interface{}{"sub_role": "Fabric"}}, nil
		case "CONFIG_DB/asic3":
			return map[string]interface{}{"DEVICE_METADATA|localhost": map[string]interface{}{"sub_role": "Weird"}}, nil // unknown, skipped
		case "CONFIG_DB/asic4":
			return nil, fmt.Errorf("boom") // error path skipped
		}
		return nil, fmt.Errorf("unexpected query %s", src)
	}
	ns, err := GetAllNamespaces(DiscardLogger, dbq)
	if err != nil {
		t.Fatalf("GetAllNamespaces error: %v", err)
	}
	if !sameStringSet(ns.Frontend, []string{"asic0"}) || !sameStringSet(ns.Backend, []string{"asic1"}) || !sameStringSet(ns.Fabric, []string{"asic2"}) {
		t.Fatalf("unexpected namespaces classification: %+v", ns)
	}
	if calls != 5 {
		t.Fatalf("expected 5 DB calls, got %d", calls)
	}
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ma := map[string]struct{}{}
	for _, s := range a {
		ma[s] = struct{}{}
	}
	for _, s := range b {
		if _, ok := ma[s]; !ok {
			return false
		}
	}
	return true
}

type fakeFileInfo struct{ name string }

func (f fakeFileInfo) Name() string       { return filepath.Base(f.name) }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0644 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() interface{}   { return nil }
