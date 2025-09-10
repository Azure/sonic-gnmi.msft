package gnmi

// Tests SHOW interface transceiver lpmode

import (
    "crypto/tls"
    "testing"
    "time"

    pb "github.com/openconfig/gnmi/proto/gnmi"
    "golang.org/x/net/context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    package gnmi

    // Tests SHOW interface transceiver lpmode

    import (
        "crypto/tls"
        "testing"
        "time"

        pb "github.com/openconfig/gnmi/proto/gnmi"
        "golang.org/x/net/context"
        "google.golang.org/grpc"
        "google.golang.org/grpc/codes"
        "google.golang.org/grpc/credentials"
    )

    func TestShowInterfaceTransceiverLpMode(t *testing.T) {
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

        patches := MockNSEnterOutput(t, "../testdata/SFPUTIL_SHOW_LPMODE_ALL.txt")
        defer patches.Reset()

        expectedAll := `[{"Port":"Ethernet0","Low-power Mode":"Off"},{"Port":"Ethernet8","Low-power Mode":"Off"},{"Port":"Ethernet16","Low-power Mode":"On"}]`
        expectedOne := `[{"Port":"Ethernet8","Low-power Mode":"Off"}]`
        expectedMissing := `[{"Port":"Ethernet24","Low-power Mode":"N/A"}]`

        tests := []struct {
            desc        string
            path        string
            wantRetCode codes.Code
            wantRespVal []byte
            valTest     bool
        }{
            {
                desc:        "query SHOW interfaces transceiver lpmode - all",
                path: `
                    elem: <name: "interfaces" >
                    elem: <name: "transceiver" >
                    elem: <name: "lpmode" >
                `,
                wantRetCode: codes.OK,
                wantRespVal: []byte(expectedAll),
                valTest:     true,
            },
            {
                desc:        "query SHOW interfaces transceiver lpmode - single existing port",
                path: `
                    elem: <name: "interfaces" >
                    elem: <name: "transceiver" >
                    elem: <name: "lpmode" key: { key: "interface" value: "Ethernet8" } >
                `,
                wantRetCode: codes.OK,
                wantRespVal: []byte(expectedOne),
                valTest:     true,
            },
            {
                desc:        "query SHOW interfaces transceiver lpmode - single missing port",
                path: `
                    elem: <name: "interfaces" >
                    elem: <name: "transceiver" >
                    elem: <name: "lpmode" key: { key: "interface" value: "Ethernet24" } >
                `,
                wantRetCode: codes.OK,
                wantRespVal: []byte(expectedMissing),
                valTest:     true,
            },
        }

        for _, tc := range tests {
            t.Run(tc.desc, func(t *testing.T) {
                runTestGet(t, ctx, gClient, "SHOW", tc.path, tc.wantRetCode, tc.wantRespVal, tc.valTest)
            })
        }
    }
                AddDataSet(t, StateDbNum, domInfoFile)
                AddDataSet(t, ConfigDbNum, configDBFile)
            },
        },
    }

    for _, test := range tests {
        if test.testInit != nil {
            test.testInit()
        }
        t.Run(test.desc, func(t *testing.T) {
            runTestGet(t, ctx, gClient, "SHOW", test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
        })
    }
}
