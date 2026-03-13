// Package dashboards embeds Grafana dashboard JSON files for provisioning.
package dashboards

import "embed"

//go:embed *.json
var FS embed.FS
