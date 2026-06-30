package mcp

const MetricSuffix = " - mcp"

// MCPMetric returns the status/Prometheus metric name for an MCP event.
func MCPMetric(base string) string {
	return base + MetricSuffix
}
