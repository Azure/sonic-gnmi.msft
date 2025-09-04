package gnmi

import (
        "os"
        "crypto/tls"
        "testing"
        "time"
        "fmt"
        "encoding/json"
        "strings"

        pb "github.com/openconfig/gnmi/proto/gnmi"
        show_client "github.com/sonic-net/sonic-gnmi/show_client"

        "github.com/agiledragon/gomonkey/v2"
        "golang.org/x/net/context"
        "google.golang.org/grpc"
        "google.golang.org/grpc/codes"
        "google.golang.org/grpc/credentials"
)

func TestGetTopMemoryUsage(t *testing.T) {
        s := createServer(t, ServerPort)
        go runServer(t, s)
        defer s.ForceStop()

        tlsConfig := &tls.Config{InsecureSkipVerify: true}
        opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}

        conn, err := grpc.Dial(TargetAddr, opts...)
        if err != nil {
                t.Fatalf("Dialing to %q failed: %v", TargetAddr, err)
        }
        defer conn.Close()

        gClient := pb.NewGNMIClient(conn)
        ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
        defer cancel()

        rawContent, err := os.ReadFile("../testdata/PROCESS_MEMORY.txt")
        if err != nil {
                t.Fatalf("Failed to read expected output file: %v", err)
        }

        expectedTopMemory, err := json.MarshalIndent(map[string]string{
                "process_memory": strings.TrimSpace(string(rawContent)),
        }, "", "  ")
        if err != nil {
                t.Fatalf("Failed to marshal expected JSON: %v", err)
        }

        ResetDataSetsAndMappings(t)

        tests := []struct {
                desc           string
                pathTarget     string
                textPbPath     string
                wantRetCode    codes.Code
                wantRespVal    interface{}
                valTest        bool
                mockOutputFile string
                testInit       func() *gomonkey.Patches
        }{
                {
                        desc:       "query show memory-usage with success case",
                        pathTarget: "SHOW",
                        textPbPath: `
                        elem: <name: "process-memory" >
                        `,
                        wantRetCode: codes.OK,
                        wantRespVal: expectedTopMemory,
                        valTest:     true,
                        mockOutputFile: "../testdata/PROCESS_MEMORY.txt",
                },
                {
                        desc:       "query show memory-usage with blank output",
                        pathTarget: "SHOW",
                        textPbPath: `
                        elem: <name: "process-memory" >
                        `,
                        wantRetCode: codes.NotFound,
                        wantRespVal: nil,
                        valTest:     false,
                        testInit: func() *gomonkey.Patches {
                                return gomonkey.ApplyFunc(show_client.GetDataFromHostCommand, func(cmd string) (string, error) {
                                        return "", nil
                                })
                        },
                },
                {
                        desc:       "query show memory-usage with error from command",
                        pathTarget: "SHOW",
                        textPbPath: `
                        elem: <name: "process-memory" >
                        `,
                        wantRetCode: codes.NotFound,
                        wantRespVal: nil,
                        valTest:     false,
                        testInit: func() *gomonkey.Patches {
                                return gomonkey.ApplyFunc(show_client.GetDataFromHostCommand, func(cmd string) (string, error) {
                                        return "", fmt.Errorf("simulated command failure")
                                })
                        },
                },
        }

        for _, test := range tests {
                var patch1, patch2 *gomonkey.Patches
                if test.testInit != nil {
                        patch1 = test.testInit()
                }

                if len(test.mockOutputFile) > 0 {
                        patch2 = MockNSEnterOutput(t, test.mockOutputFile)
                }

                t.Run(test.desc, func(t *testing.T) {
                        runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
                })

                if patch1 != nil {
                        patch1.Reset()
                }
                if patch2 != nil {
                        patch2.Reset()
                }
        }
}
