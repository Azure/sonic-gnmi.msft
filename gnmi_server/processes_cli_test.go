package gnmi

// processes_cli_test.go

// Tests SHOW processes summary|cpu|mem

import (
    "crypto/tls"
    "encoding/json"
    "testing"

    pb "github.com/openconfig/gnmi/proto/gnmi"

    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/credentials"
)


func TestGetShowProcessesVariants(t *testing.T) {
    s := createServer(t, ServerPort)
    go runServer(t, s)
    defer s.ForceStop()
    defer ResetDataSetsAndMappings(t)

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

    // Seed PROCESS_STATS data
    FlushDataSet(t, StateDbNum)
    AddDataSet(t, StateDbNum, "../testdata/PROCESS_STATS_SAMPLE.txt")

    tests := []struct {
        desc         string
        textPbPath   string
        wantRetCode  codes.Code
        wantRespJson string
    }{
        {
            desc: "processes root help",
            textPbPath: `
                elem: <name: "processes" >
            `,
            wantRetCode:  codes.OK,
            // We'll validate structure rather than exact ordering below when wantRespJson == "__HELP__"
            wantRespJson: "__HELP__",
        },
        {
            desc: "processes summary pid order asc",
            textPbPath: `
                elem: <name: "processes" >
                elem: <name: "summary" >
            `,
            wantRetCode:  codes.OK,
            wantRespJson: `[{"PID":"123","PPID":"1","CMD":"redis-server","%MEM":"1.2","%CPU":"0.5"},{"PID":"456","PPID":"1","CMD":"swss","%MEM":"3.4","%CPU":"15.0"},{"PID":"789","PPID":"456","CMD":"orchagent","%MEM":"2.0","%CPU":"7.5"}]`,
        },
        {
            desc: "processes cpu order desc then pid",
            textPbPath: `
                elem: <name: "processes" >
                elem: <name: "cpu" >
            `,
            wantRetCode:  codes.OK,
            wantRespJson: `[{"PID":"456","PPID":"1","CMD":"swss","%MEM":"3.4","%CPU":"15.0"},{"PID":"789","PPID":"456","CMD":"orchagent","%MEM":"2.0","%CPU":"7.5"},{"PID":"123","PPID":"1","CMD":"redis-server","%MEM":"1.2","%CPU":"0.5"}]`,
        },
        {
            desc: "processes mem order desc then pid",
            textPbPath: `
                elem: <name: "processes" >
                elem: <name: "mem" >
            `,
            wantRetCode:  codes.OK,
            wantRespJson: `[{"PID":"456","PPID":"1","CMD":"swss","%MEM":"3.4","%CPU":"15.0"},{"PID":"789","PPID":"456","CMD":"orchagent","%MEM":"2.0","%CPU":"7.5"},{"PID":"123","PPID":"1","CMD":"redis-server","%MEM":"1.2","%CPU":"0.5"}]`,
        },
    }

    for _, tc := range tests {
        t.Run(tc.desc, func(t *testing.T) {
            resp := runTestGet(t, ctx, gClient, "SHOW", tc.textPbPath, tc.wantRetCode, nil, false)
            if tc.wantRetCode != codes.OK {
                return
            }
            gotJson := string(resp.GetNotification()[0].Update[0].Val.GetJsonIetfVal())
            if tc.wantRespJson == "__HELP__" {
                // Validate help structure: must contain subcommands summary,cpu,mem
                var m map[string]interface{}
                if err := json.Unmarshal([]byte(gotJson), &m); err != nil {
                    t.Fatalf("failed to unmarshal help json: %v", err)
                }
                scAny, ok := m["subcommands"]
                if !ok {
                    t.Fatalf("help json missing subcommands: %v", gotJson)
                }
                sc, ok := scAny.(map[string]interface{})
                if !ok {
                    t.Fatalf("subcommands not an object: %T", scAny)
                }
                for _, k := range []string{"summary", "cpu", "mem"} {
                    if _, ok := sc[k]; !ok {
                        t.Fatalf("subcommand %s missing in help json: %v", k, gotJson)
                    }
                }
            } else {
                if gotJson != tc.wantRespJson {
                    t.Fatalf("response mismatch\n got : %s\n want: %s", gotJson, tc.wantRespJson)
                }
            }
        })
    }
}
