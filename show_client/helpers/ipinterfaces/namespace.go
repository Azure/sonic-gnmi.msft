package ipinterfaces

import (
	"fmt"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

const (
	asicNamePrefix   = "asic"
	defaultNamespace = ""
)

// GetAllNamespaces returns a slice of all network namespace names.
// On a single-ASIC system, it returns a slice with one empty string ""
// which represents the default (host) namespace.
func GetAllNamespaces() (*NamespacesByRole, error) {
	numAsics := common.ReadAsicConfValue()
	if numAsics <= 1 {
		// Single ASIC platform, return the default namespace in the frontend role.
		return &NamespacesByRole{Frontend: []string{defaultNamespace}}, nil
	}

	// Multi-ASIC platform, discover namespaces and their roles from ConfigDB.
	namespaces := NamespacesByRole{}
	for i := 0; i < numAsics; i++ {
		ns := fmt.Sprintf("%s%d", asicNamePrefix, i)
		dbTarget := fmt.Sprintf("CONFIG_DB/%s", ns)
		queries := [][]string{{dbTarget, "DEVICE_METADATA", "localhost"}}

		msi, err := common.GetMapFromQueries(queries)
		if err != nil {
			// Log warning but continue, one failing namespace shouldn't stop the whole process.
			log.Warningf("could not get metadata for namespace '%s': %v", ns, err)
			continue
		}

		key := "DEVICE_METADATA|localhost"
		entry, ok := msi[key].(map[string]interface{})
		if !ok {
			log.Warningf("could not parse metadata for namespace '%s'", ns)
			continue
		}

		if subRole, ok := entry["sub_role"].(string); ok {
			switch subRole {
			case "Frontend":
				namespaces.Frontend = append(namespaces.Frontend, ns)
			case "Backend":
				namespaces.Backend = append(namespaces.Backend, ns)
			case "Fabric":
				namespaces.Fabric = append(namespaces.Fabric, ns)
			}
		}
	}

	return &namespaces, nil
}
