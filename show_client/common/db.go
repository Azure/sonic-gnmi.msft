package common

import (
	"fmt"

	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	dbIndex    = 0 // The first index for a query will be the DB
	tableIndex = 1 // The second index for a query will be the table

	minQueryLength = 2 // We need to support TARGET/TABLE as a minimum query
	maxQueryLength = 5 // We can support up to 5 elements in query (TARGET/TABLE/(2 KEYS)/FIELD)
)

const (
	StateDb    = "STATE_DB"
	ConfigDb   = "CONFIG_DB"
	ApplDb     = "APPL_DB"
	CountersDb = "COUNTERS_DB"

	ConfigDBPortTable        = "PORT"
	AppDBPortTable           = "PORT_TABLE"
	StateDBPortTable         = "PORT_TABLE"
	ConfigDBPortChannelTable = "PORTCHANNEL"
	AppDBPortChannelTable    = "LAG_TABLE"
	FDBTable                 = "FDB_TABLE"
)

func GetMapFromQueries(queries [][]string) (map[string]interface{}, error) {
	tblPaths, err := CreateTablePathsFromQueries(queries)
	if err != nil {
		return nil, err
	}
	msi := make(map[string]interface{})
	for _, tblPath := range tblPaths {
		err := sdc.TableData2Msi(&tblPath, false, nil, &msi)
		if err != nil {
			return nil, err
		}
	}
	return msi, nil
}

func GetDataFromQueries(queries [][]string) ([]byte, error) {
	msi, err := GetMapFromQueries(queries)
	if err != nil {
		return nil, err
	}
	return sdc.Msi2Bytes(msi)
}

func CreateTablePathsFromQueries(queries [][]string) ([]sdc.TablePath, error) {
	var allPaths []sdc.TablePath

	// Create and validate gnmi path then create table path
	for _, q := range queries {
		queryLength := len(q)
		if queryLength < minQueryLength || queryLength > maxQueryLength {
			return nil, fmt.Errorf("invalid query %v: must support at least [DB, table] or at most [DB, table, key1, key2, field]", q)
		}

		// Build a gNMI path for validation:
		//   prefix = { Target: dbTarget }
		//   path   = { Elem: [ {Name:table}, {Name:key}, {Name:field} ] }

		dbTarget := q[dbIndex]
		prefix := &gnmipb.Path{Target: dbTarget}

		table := q[tableIndex]
		elems := []*gnmipb.PathElem{{Name: table}}

		// Additional elements like keys and fields
		for i := tableIndex + 1; i < queryLength; i++ {
			elems = append(elems, &gnmipb.PathElem{Name: q[i]})
		}

		path := &gnmipb.Path{Elem: elems}

		if tablePaths, err := sdc.PopulateTablePaths(prefix, path); err != nil {
			return nil, fmt.Errorf("query %v failed: %w", q, err)
		} else {
			allPaths = append(allPaths, tablePaths...)
		}
	}
	return allPaths, nil
}
