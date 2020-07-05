package main

import (
	"flag"
	"fmt"
	"html/template"
	"math/rand"
	"nodemanager"
	"os"
	"time"
)

type arrayFlag []string

var (
	majorVersion = 1
	minorVersion = 0
	header       = map[string]interface{}{
		"Name":      "NAME",
		"CreatedAt": "CREATED",
		"UpdatedAt": "UPDATED",
		"Labels":    "LABELS",
	}
	formatFlag     string
	envFlag        arrayFlag
	capsFlag       arrayFlag
	nameFlag       string
	privilegedFlag bool
	workingDirFlag string
)

func (i *arrayFlag) String() string {
	return "my string representation"
}

func (i *arrayFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString(length int) string {
	return stringWithCharset(length, charset)
}

func helpRuntime() {
	fmt.Printf("NodeManager CLI - Version %d.%d", majorVersion, minorVersion)
	fmt.Println("Runtime available actions:")
	fmt.Println("\tlist\t: prints a list of the existing runtimes")
	fmt.Println("\tremove\t: removes a existing runtime")
	fmt.Println("\tpull\t: pulls a runtime from the remote server")
	os.Exit(0)
}

func helpNode() {
	fmt.Printf("NodeManager CLI - Version %d.%d", majorVersion, minorVersion)
	fmt.Println("Node available actions:")
	fmt.Println("\tlist\t: list available nodes")
	fmt.Println("\tcreate\t: create a new node")
	fmt.Println("\tdestroy\t: destroy a node")
	os.Exit(0)
}

func helpTask() {
	fmt.Printf("NodeManager CLI - Version %d.%d", majorVersion, minorVersion)
	fmt.Println("Task available actions:")
	fmt.Println("\tlist\t: list running nodes")
	fmt.Println("\trun\t: run a new task at node")
	fmt.Println("\tsignal\t: send a signal to a task")
	os.Exit(0)
}

func help() {
	fmt.Printf("NodeManager CLI - Version %d.%d", majorVersion, minorVersion)
	fmt.Println("Available objects:")
	fmt.Println("\truntime")
	fmt.Println("\tnode")
	os.Exit(0)
}

func runtimeList() {

	format := formatFlag

	if len(format) == 0 {
		format = "{{.Name}} {{.UpdatedAt}}\n"
	}

	tmpl, err := template.New("nodemanagercli").Parse(format)

	if err != nil {
		panic(err)
	}

	nm := nodemanager.GetNodeManager()
	runtimes, err := nm.RuntimeList()

	if err != nil {
		panic(err)
	}

	if runtimes == nil {
		return
	}

	for i := 0; i < len(runtimes); i++ {
		tmpl.Execute(os.Stdout, runtimes[i])
	}
}

func runtimeRemove(args []string) {

	if len(args) < 3 {
		panic("Missing runtime name")
	}

	fmt.Printf("Removing Runtime:%q\n", args[2])
	nm := nodemanager.GetNodeManager()
	nm.RuntimeDelete(args[2])
}

func runtimePull(args []string) {

	if len(args) < 3 {
		panic("Missing runtime name")
	}

	fmt.Printf("Pulling Runtime:%q\n", args[2])
	nm := nodemanager.GetNodeManager()
	nm.RuntimePull(args[2])
}

func nodeList() {
	format := formatFlag

	if len(format) == 0 {
		format = "{{.ID}} {{.Image}} {{.UpdatedAt}}\n"
	}

	tmpl, err := template.New("nodemanagercli").Parse(format)

	if err != nil {
		panic(err)
	}

	nm := nodemanager.GetNodeManager()
	nodes, err := nm.NodeList()
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(nodes); i++ {
		tmpl.Execute(os.Stdout, nodes[i])
	}
}

func nodeCreate(args []string) {

	if len(args) < 3 {
		panic("Missing runtime name")
	}

	var node nodemanager.NodeSpec

	node.Caps = capsFlag
	node.Env = envFlag
	node.Runtime = args[2]
	node.Name = nameFlag
	node.Privileged = privilegedFlag

	if len(node.Name) == 0 {
		node.Name = randomString(10)
	}

	nm := nodemanager.GetNodeManager()

	nm.NodeCreate(&node)
}

func nodeDestroy(args []string) {

	if len(args) < 3 {
		panic("Missing runtime name")
	}

	nm := nodemanager.GetNodeManager()

	nm.NodeDestroy(args[2])
}

func taskList() {

	format := formatFlag

	if len(format) == 0 {
		format = "{{.Pid}} {{.ID}} {{.ContainerID}} {{.Status}}\n"
	}

	tmpl, err := template.New("nodemanagercli").Parse(format)

	if err != nil {
		panic(err)
	}

	nm := nodemanager.GetNodeManager()
	tasks, err := nm.TaskList()

	if err != nil {
		panic(err)
	}

	for i := 0; i < len(tasks.Tasks); i++ {
		tmpl.Execute(os.Stdout, tasks.Tasks[i])
	}
}

func taskRun(args []string) {

	if len(args) < 4 {
		panic("Missing parameters")
	}

	var task nodemanager.TaskSpec

	task.Name = nameFlag
	task.Args = args[3:]
	task.Env = envFlag
	task.Node.Name = args[2]
	task.WorkingDir = workingDirFlag

	if len(task.WorkingDir) == 0 {
		task.WorkingDir = "/"
	}

	nm := nodemanager.GetNodeManager()

	err := nm.TaskExec(&task, os.Stdin, os.Stdout, os.Stderr)

	if err != nil {
		panic(err)
	}
}

func taskSignal(args []string) {

	fmt.Println("Task Signal")
}

func runtimeAction(args []string) {

	if len(args) < 2 {
		helpRuntime()
	}

	switch args[1] {
	case "list":
		runtimeList()
		break
	case "remove":
		runtimeRemove(args)
		break
	case "pull":
		runtimePull(args)
		break
	default:
		help()
		break
	}

}

func nodeActions(args []string) {

	if len(args) < 2 {
		helpNode()
	}

	switch args[1] {
	case "list":
		nodeList()
		break
	case "create":
		nodeCreate(args)
		break
	case "destroy":
		nodeDestroy(args)
		break
	default:
		helpNode()
		break
	}
}

func taskActions(args []string) {

	if len(args) < 2 {
		helpTask()
	}

	switch args[1] {
	case "list":
		taskList()
		break
	case "run":
		taskRun(args)
		break
	case "signal":
		taskSignal(args)
		break
	default:
		helpTask()
		break
	}
}

func main() {
	flag.BoolVar(&privilegedFlag, "privileged", false, "Print format")
	flag.StringVar(&workingDirFlag, "workingdir", "", "Print format")
	flag.StringVar(&formatFlag, "format", "", "Print format")
	flag.Var(&envFlag, "env", "Environment Variables")
	flag.Var(&capsFlag, "caps", "Environment Variables")
	flag.StringVar(&nameFlag, "name", "", "Environment Variables")
	flag.Parse()
	if len(flag.Args()) < 1 {
		help()
	}
	switch flag.Arg(0) {
	case "runtime":
		runtimeAction(flag.Args())
		break
	case "node":
		nodeActions(flag.Args())
		break
	case "task":
		taskActions(flag.Args())
		break
	default:
		help()
		break
	}
}
