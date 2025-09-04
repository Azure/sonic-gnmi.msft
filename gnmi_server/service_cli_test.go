package gnmi

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

func TestGetServices(t *testing.T) {
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

	cmdAndFileMapping := map[string]string{
		"{{.Names}}": "../testdata/dockerProcessNamesOutput.txt",
		"telemetry":  "../testdata/telemetryProcessInfo.txt",
		"swss":       "../testdata/swssProcessInfo.txt",
	}

	patches := MockExecCmds(t, cmdAndFileMapping)

	t.Run("query SHOW services", func(t *testing.T) {
		textPbPath := `
			elem: <name: "services" >
		`
		wantRespVal := []byte(`
			[
				{
					"dockerProcessName": "telemetry",
					"processes": [
						{
							"command": "/usr/bin/python3 /usr/local/bin/supervisord",
							"cpuPercentage": "1.3",
							"memPercentage": "0.1",
							"pid": "1",
							"rss": "26240",
							"start": "07:57",
							"stat": "Ss+",
							"time": "0:00",
							"tty": "pts/0",
							"user": "root",
							"vsz": "30544"
						},
						{
							"command": "python3 /usr/bin/supervisor-proc-exit-listener --container-name telemetry",
							"cpuPercentage": "0.3",
							"memPercentage": "0.1",
							"pid": "8",
							"rss": "27540",
							"start": "07:57",
							"stat": "Sl",
							"time": "0:00",
							"tty": "pts/0",
							"user": "root",
							"vsz": "124964"
						},
						{
							"command": "/usr/sbin/rsyslogd -n -iNONE",
							"cpuPercentage": "0.0",
							"memPercentage": "0.0",
							"pid": "11",
							"rss": "4236",
							"start": "07:57",
							"stat": "Sl",
							"time": "0:00",
							"tty": "pts/0",
							"user": "root",
							"vsz": "222220"
						}
					]
				},
				{
					"dockerProcessName": "swss",
					"processes": [
						{
							"command": "python3 /usr/bin/supervisor-proc-exit-listener --container-name swss",
							"cpuPercentage": "0.0",
							"memPercentage": "0.1",
							"pid": "41",
							"rss": "27544",
							"start": "08:32",
							"stat": "Sl",
							"time": "0:00",
							"tty": "pts/0",
							"user": "root",
							"vsz": "124992"
						},
						{
							"command": "/usr/sbin/rsyslogd -n -iNONE",
							"cpuPercentage": "0.0",
							"memPercentage": "0.0",
							"pid": "44",
							"rss": "4308",
							"start": "08:32",
							"stat": "Sl",
							"time": "0:00",
							"tty": "pts/0",
							"user": "root",
							"vsz": "230436"
						},
						{
							"command": "/usr/bin/orchagent -d /var/log/swss -b 1024 -s -m b0:8d:57:f5:b3:60",
							"cpuPercentage": "0.2",
							"memPercentage": "0.2",
							"pid": "58",
							"rss": "34668",
							"start": "08:32",
							"stat": "Sl",
							"time": "0:00",
							"tty": "pts/0",
							"user": "root",
							"vsz": "567220"
						},
						{
							"command": "/usr/bin/rsyslog_plugin -r /etc/rsyslog.d/swss_regex.json -m sonic-events-swss",
							"cpuPercentage": "0.0",
							"memPercentage": "0.0",
							"pid": "93",
							"rss": "10836",
							"start": "08:32",
							"stat": "Sl",
							"time": "0:00",
							"tty": "pts/0",
							"user": "root",
							"vsz": "100860"
						},
						{
							"command": "/usr/bin/buffermgrd -l /usr/share/sonic/hwsku/pg_profile_lookup.ini",
							"cpuPercentage": "0.0",
							"memPercentage": "0.0",
							"pid": "110",
							"rss": "10988",
							"start": "08:32",
							"stat": "Sl",
							"time": "0:00",
							"tty": "pts/0",
							"user": "root",
							"vsz": "92544"
						}
					]
				}
			]
		`)
		runTestGet(t, ctx, gClient, "SHOW", textPbPath, codes.OK, wantRespVal, true)
	})

	patches.Reset()
}
