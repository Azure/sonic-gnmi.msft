package gnmi

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	sdcfg "github.com/sonic-net/sonic-gnmi/sonic_db_config"
)

const (
	ServerPort    = 8081
	ApplDbNum     = 0
	AsicDbNum     = 1
	CountersDbNum = 2
	ConfigDbNum   = 4
	StateDbNum    = 6

	TargetAddr        = "127.0.0.1:8081"
	QueryTimeout      = 10
	ChassisStateDbNum = 13
)

func MockNSEnterOutput(t *testing.T, fileName string) *gomonkey.Patches {
	fileContentBytes, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("read file %v err: %v", fileName, err)
	}
	patches := gomonkey.ApplyFunc(exec.Command, func(name string, args ...string) *exec.Cmd {
		return &exec.Cmd{}
	})
	patches.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "CombinedOutput", func(_ *exec.Cmd) ([]byte, error) {
		return fileContentBytes, nil
	})
	return patches
}

func MockNSEnterCommand(t *testing.T, fileName string) *gomonkey.Patches {
	fileContentBytes, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("read file %v err: %v", fileName, err)
	}
	patches := gomonkey.ApplyFunc(exec.Command, func(name string, args ...string) *exec.Cmd {
		return &exec.Cmd{}
	})
	patches.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "CombinedOutput", func(_ *exec.Cmd) ([]byte, error) {
		return fileContentBytes, nil
	})
	return patches
}

func MockExecCmds(t *testing.T, cmdAndFileMap map[string]string) *gomonkey.Patches {
	var fileContentBytes []byte
	var err error

	patches := gomonkey.ApplyFunc(exec.Command, func(name string, args ...string) *exec.Cmd {
		found := false
		for mockCmd := range cmdAndFileMap {
			for _, cmdArg := range args {
				if strings.Contains(cmdArg, mockCmd) {
					fileContentBytes, err = os.ReadFile(cmdAndFileMap[mockCmd])
					if err != nil {
						t.Fatalf("read file %v err: %v", cmdAndFileMap[mockCmd], err)
					}
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		return &exec.Cmd{}
	})
	patches.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "CombinedOutput", func(_ *exec.Cmd) ([]byte, error) {
		return fileContentBytes, nil
	})
	return patches
}

func MockNSEnterCommandError(commandErr error) *gomonkey.Patches {
	patches := gomonkey.ApplyFunc(exec.Command, func(name string, args ...string) *exec.Cmd {
		return &exec.Cmd{}
	})
	patches.ApplyMethod(reflect.TypeOf(&exec.Cmd{}), "CombinedOutput", func(_ *exec.Cmd) ([]byte, error) {
		return nil, commandErr
	})
	return patches
}

func MockReadFile(fileName string, fileContent string, fileReadErr error) {
	sdc.ImplIoutilReadFile = func(filePath string) ([]byte, error) {
		if filePath == fileName {
			if fileReadErr != nil {
				return nil, fileReadErr
			}
			return []byte(fileContent), nil
		}
		return os.ReadFile(filePath)
	}
}

func FlushDataSet(t *testing.T, dbNum int) {
	ns, _ := sdcfg.GetDbDefaultNamespace()
	rclient := getRedisClientN(t, dbNum, ns)
	defer rclient.Close()
	rclient.FlushDB(context.Background())
}

func AddDataSet(t *testing.T, dbNum int, fileName string) {
	sdc.ClearMappings()
	LoadDataSet(t, dbNum, fileName)
}

// LoadDataSet loads data into Redis without clearing in-memory mappings.
// Use this when only data values change (e.g. counter/rate values) but
// port-to-OID and alias mappings remain the same.
func LoadDataSet(t *testing.T, dbNum int, fileName string) {
	ns, _ := sdcfg.GetDbDefaultNamespace()
	rclient := getRedisClientN(t, dbNum, ns)
	defer rclient.Close()

	fileContentBytes, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("read file: %v, err: %v", fileName, err)
	}

	fileContent := loadConfig(t, "", fileContentBytes)
	loadDB(t, rclient, fileContent)
}

func ResetDataSetsAndMappings(t *testing.T) {
	FlushDataSet(t, ApplDbNum)
	FlushDataSet(t, AsicDbNum)
	FlushDataSet(t, CountersDbNum)
	FlushDataSet(t, ConfigDbNum)
	FlushDataSet(t, StateDbNum)
	FlushDataSet(t, ChassisStateDbNum)
	sdc.ClearMappings()
}

func MockEnvironmentVariable(t *testing.T, key string, value string) {
	t.Setenv(key, value)
}
