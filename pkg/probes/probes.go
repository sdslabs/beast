package probes

type ProbeResult string

const (
	Success ProbeResult = "success"
	Warning ProbeResult = "warning"
	Failure ProbeResult = "failure"
	Unknown ProbeResult = "unknown"
)
