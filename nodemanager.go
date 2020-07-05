// Copyright 2020 Alexandre Pires (c.alexandre.pires@gmail.com)

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nodemanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// NodeManager the node manager implementation
type NodeManager struct {
	Logger *log.Logger
	Client *containerd.Client
}

// logger default logger

var (
	defaultNamespace = "nodemanager"
	nodeManagerSync  sync.Once
	nodeManager      *NodeManager
)

// RuntimeExists check if runtime exits
func (manager *NodeManager) RuntimeExists(name string) bool {

	if manager.Client == nil {
		return false
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)
	_, err := manager.Client.ImageService().Get(ctx, name)

	return err == nil
}

// RuntimePull pull a runtime from a store
func (manager *NodeManager) RuntimePull(name string) error {

	if manager.Client == nil {

		return errors.New("Containerd available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	_, err := manager.Client.Pull(ctx, name, containerd.WithPullUnpack)

	if err != nil {

		return err
	}

	return nil
}

// RuntimeDelete delete a local runtime
func (manager *NodeManager) RuntimeDelete(name string) error {

	if manager.Client == nil {

		return errors.New("Containerd available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	err := manager.Client.ImageService().Delete(ctx, name)

	return err
}

// RuntimeList list the available runtimes
func (manager *NodeManager) RuntimeList() ([]images.Image, error) {

	if manager.Client == nil {

		return nil, errors.New("Containerd available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)
	return manager.Client.ImageService().List(ctx)
}

// NodeCreate create a node
func (manager *NodeManager) NodeCreate(node *NodeSpec) error {

	if manager.Client == nil {

		return errors.New("Containerd available")
	}

	var runtime containerd.Image
	var err error

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	if manager.NodeExists(node.Name) {

		return errors.New("NodeSpec already exists")
	}

	if !manager.RuntimeExists(node.Runtime) {

		runtime, err = manager.Client.Pull(ctx, node.Runtime, containerd.WithPullUnpack)

		if err != nil {

			return err
		}

	} else {

		runtime, err = manager.Client.GetImage(ctx, node.Runtime)

		if err != nil {

			return err
		}
	}

	// TODO: add more specs like devices, env, etc
	newOpts := containerd.WithNewSpec(
		oci.WithImageConfig(runtime),
		oci.WithEnv(node.Env),
		oci.WithCapabilities(node.Caps),
	)

	snapShotname := node.Name

	if node.Privileged {

		_, err = manager.Client.NewContainer(
			ctx,
			node.Name,
			containerd.WithNewSnapshot(snapShotname, runtime),
			newOpts,
			containerd.WithNewSpec(oci.WithPrivileged),
		)
	} else {
		_, err = manager.Client.NewContainer(
			ctx,
			node.Name,
			containerd.WithNewSnapshot(snapShotname, runtime),
			newOpts,
		)
	}

	return err
}

// NodeDestroy destroy node
func (manager *NodeManager) NodeDestroy(name string) error {

	if manager.Client == nil {

		return errors.New("Containerd not available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := manager.Client.LoadContainer(ctx, name)

	if err != nil {

		return err
	}

	// Force a kill request of the associated task
	_, err = manager.Client.TaskService().Kill(ctx, &tasks.KillRequest{
		ContainerID: container.ID(),
		Signal:      9,
		All:         true,
	})

	// Force delete the associated task
	_, err = manager.Client.TaskService().Delete(ctx, &tasks.DeleteTaskRequest{
		ContainerID: container.ID(),
	})

	err = container.Delete(ctx, containerd.WithSnapshotCleanup)

	return err
}

// NodeList list existing Nodes
func (manager *NodeManager) NodeList() ([]containers.Container, error) {

	if manager.Client == nil {

		return nil, errors.New("Containerd not available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	return manager.Client.ContainerService().List(ctx)
}

// NodeExists check if node exists
func (manager *NodeManager) NodeExists(name string) bool {

	if manager.Client == nil {

		return false
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	_, err := manager.Client.ContainerService().Get(ctx, name)

	return err == nil
}

// NodeLoad load an existing node
func (manager *NodeManager) NodeLoad(name string) (*NodeSpec, error) {

	if manager.Client == nil {

		return nil, errors.New("Containerd available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := manager.Client.ContainerService().Get(ctx, name)

	if err != nil {

		return nil, err
	}

	runtime := &NodeSpec{
		Name:    container.ID,
		Runtime: container.Image,
		// TODO: get missing properties
	}

	return runtime, nil
}

// MountList list all mounts
func (manager *NodeManager) MountList() ([]mount.Mount, error) {

	if manager.Client == nil {
		return nil, errors.New("Containerd not available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	return manager.Client.SnapshotService(containerd.DefaultSnapshotter).Mounts(ctx, "")
}

// TaskExec execute a task inside a node
func (manager *NodeManager) TaskExec(task *TaskSpec, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {

	if manager.Client == nil {
		return errors.New("Containerd not available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := manager.Client.LoadContainer(ctx, task.Node.Name)

	if err != nil {
		return err
	}

	ciOptions := cio.NewCreator(
		cio.WithStreams(stdin, stdout, stderr),
	)

	if task.Terminal {
		ciOptions = cio.NewCreator(
			cio.WithStreams(os.Stdin, os.Stdout, os.Stderr),
		)
	}

	fmt.Println(task.Name)
	taskEnv, err := container.NewTask(ctx, ciOptions)

	if err != nil {

		return err
	}

	_, err = taskEnv.Wait(ctx)
	if err != nil {

		return err
	}

	execOpts := &specs.Process{
		Args: task.Args,
		Env:  task.Env,
		Cwd:  task.WorkingDir,
	}

	process, err := taskEnv.Exec(ctx, task.Name, execOpts, ciOptions)

	if err != nil {

		return err
	}

	err = process.Start(ctx)

	if err != nil {

		return err
	}

	_, err = process.Wait(ctx)
	if err != nil {

		return err
	}

	task.PID = process.Pid()

	return nil
}

// TaskList get a list of tasks
func (manager *NodeManager) TaskList() (*tasks.ListTasksResponse, error) {

	if manager.Client == nil {
		return nil, errors.New("Containerd not available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	var tr tasks.ListTasksRequest

	return manager.Client.TaskService().List(ctx, &tr)
}

// SignalEmit send a signal to a node
func (manager *NodeManager) SignalEmit(signal *SignalSpec) error {

	if manager.Client == nil {
		return errors.New("Containerd not available")
	}

	ctx := namespaces.WithNamespace(context.Background(), defaultNamespace)

	container, err := manager.Client.LoadContainer(ctx, signal.Node)

	if err != nil {
		return err
	}

	ciAttach := cio.NewAttach(
		cio.WithStreams(nil, nil, nil),
	)

	taskEnv, err := container.Task(ctx, ciAttach)
	if err != nil {

		return err
	}

	process, err := taskEnv.LoadProcess(ctx, signal.Task, ciAttach)
	if err != nil {

		return err
	}

	return process.Kill(ctx, signal.Signal)
}

// ProcessEnvelope Envelope processor
func (manager *NodeManager) ProcessEnvelope(reader io.Reader, writer io.Writer) error {

	var envelope EnvelopeSpec

	decoder := json.NewDecoder(reader)
	encoder := json.NewEncoder(writer)

	err := decoder.Decode(&envelope)

	if err != nil {

		return err
	}

	switch envelope.Kind {
	case "NodeSpec":
		var node NodeSpec
		if err := json.Unmarshal(envelope.Message, &node); err != nil {

			return err
		}
		switch envelope.Action {
		case "create":
			err := manager.NodeCreate(&node)
			if err == nil {
				encoder.Encode(&node)
			}
			return err
		case "destroy":
			err := manager.NodeDestroy(node.Name)
			if err == nil {
				encoder.Encode(&node)
			}
			return err
		default:

			return errors.New("Invalid action")
		}
	case "TaskSpec":
		var task TaskSpec
		if err := json.Unmarshal(envelope.Message, &task); err != nil {

			return err
		}
		switch envelope.Action {
		case "run":
			err := manager.TaskExec(&task, nil, manager.Logger.Writer(), manager.Logger.Writer())
			if err == nil {
				encoder.Encode(&task)
			}
			return err
		default:

			return errors.New("Invalid action")
		}
	case "SignalSpec":
		var signal SignalSpec
		if err := json.Unmarshal(envelope.Message, &signal); err != nil {

			return err
		}
		switch envelope.Action {
		case "emit":
			err := manager.SignalEmit(&signal)
			if err == nil {
				encoder.Encode(&signal)
			}
			return err
		default:

			return errors.New("Invalid envelope")
		}
	default:

		return errors.New("Invalid envelope")
	}
}

// GetNodeManager Initialize modules compoment
func GetNodeManager() *NodeManager {

	var err error

	nodeManagerSync.Do(func() {

		nodeManager = &NodeManager{}
		// Connect to the containerd socket
		nodeManager.Client, err = containerd.New("/run/containerd/containerd.sock")

		if err != nil {
			panic(err)
		}
	})

	if nodeManager == nil {
		panic("NodeManager: Something wen't wrong when starting NodeSpec manager!")
	}

	return nodeManager
}
