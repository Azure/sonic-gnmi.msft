package metadata

import (
	"os"
)

var metadataContainerVersion string = DefaultVersion

func init() {
	if containerVersion := os.Getenv(VersionEnvVar); containerVersion != "" {
		metadataContainerVersion = containerVersion
	}
}

func Version() string { return metadataContainerVersion }

func SetVersionTest(version string) {
	value := os.Getenv("UNIT_TEST")
	if value != "1" {
		return
	}
	metadataContainerVersion = version
}
