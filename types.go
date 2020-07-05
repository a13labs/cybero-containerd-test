package nodemanager

import (
	"encoding/json"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// NodeSpec Node specification
type NodeSpec struct {
	Name       string
	Runtime    string
	Env        []string
	Devices    []specs.LinuxDevice
	Caps       []string
	Privileged bool
	GIDs       string
}

// TaskSpec Task specification
type TaskSpec struct {
	Node       NodeSpec
	Name       string
	Args       []string
	Env        []string
	WorkingDir string
	PID        uint32
	Terminal   bool
}

// SignalSpec Job specification
type SignalSpec struct {
	Node   string
	Task   string
	Signal syscall.Signal
}

// EnvelopeSpec Envelope specification
type EnvelopeSpec struct {
	Kind    string          `json:"kind"`
	Action  string          `json:"action"`
	Message json.RawMessage `json:"message"`
}
