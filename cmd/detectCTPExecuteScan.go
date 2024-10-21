package cmd

import "github.com/SAP/jenkins-library/pkg/telemetry"

func detectCTPExecuteScan(config detectCTPExecuteScanOptions, telemetryData *telemetry.CustomData, influx *detectCTPExecuteScanInflux) {
	detectConfig := detectExecuteScanOptions(config)
	detectInflux := detectExecuteScanInflux(*influx)

	detectExecuteScan(detectConfig, telemetryData, &detectInflux)
}
