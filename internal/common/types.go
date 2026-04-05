package common

import "time"

// PodEvent represents a normalized pod lifecycle event captured from Kubernetes.
type PodEvent struct {
	Type      string            `json:"type"`
	Namespace string            `json:"namespace"`
	Pod       string            `json:"pod"`
	Reason    string            `json:"reason"`
	Message   string            `json:"message"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// ThreatAlert is produced by the analyzer and consumed by the alert manager.
type ThreatAlert struct {
	Severity  string    `json:"severity"`
	Category  string    `json:"category"`
	Source    string    `json:"source"`
	Namespace string    `json:"namespace"`
	Pod       string    `json:"pod"`
	Summary   string    `json:"summary"`
	Detected  time.Time `json:"detected"`
}
