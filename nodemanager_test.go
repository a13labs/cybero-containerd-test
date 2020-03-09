package nodemanager

import (
	"log"
	"os"
	"syscall"
	"testing"
	"time"
)

// TestNodeManager Test NodeManage implementation
func TestNodeManager(t *testing.T) {

	logFile, err := os.OpenFile("/tmp/nodemanager-test.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	defer logFile.Close()

	// Initialize logging to a file
	if err != nil {
		t.Errorf("Error opening logfile, err: %v\n", err)
		return
	}

	logger := log.New(logFile, "", log.LstdFlags)
	nodeManager := GetNodeManager(logger)

	if nodeManager == nil {
		t.Errorf("No nodemanager instance\n")
		return
	}

	nodeManager.RuntimePull("docker.io/library/redis:alpine")

	var node *NodeSpec

	nodeName := "redis-manager-test"
	nodeRuntime := "docker.io/library/redis:alpine"

	if !nodeManager.NodeExists(nodeName) {

		node = &NodeSpec{
			Name:    nodeName,
			Runtime: nodeRuntime,
		}

		err = nodeManager.NodeCreate(node)

		if err != nil {
			t.Errorf("Error creating node, err: %v\n", err)
		}

	}

	node, err = nodeManager.NodeLoad(nodeName)

	if err != nil {
		t.Errorf("Error loading existing node!, err: %v\n", err)
	}

	task := &TaskSpec{
		Node:       *node,
		Name:       "command",
		Args:       []string{"/usr/local/bin/redis-server", "--port 7777"},
		WorkingDir: "/",
		Env:        []string{"PYTHONPATH=/usr/bin"},
	}

	err = nodeManager.TaskExec(task, os.Stdin, os.Stdout, os.Stderr)

	if err != nil {
		t.Errorf("Error creating task")
	}

	signal := &SignalSpec{
		Node:   node.Name,
		Task:   task.Name,
		Signal: syscall.SIGTERM,
	}

	time.Sleep(5 * time.Second)
	err = nodeManager.SignalEmit(signal)

	if err != nil {
		t.Errorf("Error sending signal to task, err: %v\n", err)
	}

	err = nodeManager.NodeDestroy(node)

	if err != nil {
		t.Errorf("Error destroying runtime, err: %v\n", err)
	}

}
