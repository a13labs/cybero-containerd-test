package main

import (
	"fmt"
	"nodemanager"
	"os"
)

func main() {

	if len(os.Args) == 1 {
		fmt.Println("File is missing, aborting....")
		os.Exit(1)
	}

	file, err := os.Open(os.Args[1])
	defer file.Close()
	if err != nil {
		fmt.Printf("Error opening %q\n", os.Args[1])
		os.Exit(1)
	}

	nodeManager := nodemanager.GetNodeManager(nil)
	err = nodeManager.ProcessEnvelope(file, os.Stdout)

	if err != nil {
		fmt.Printf("nmcli: Error processing envelope, err: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
