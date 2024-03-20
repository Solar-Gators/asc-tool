//create a command line interface for the user to interact with the application

package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// create a new flag set
	fs := flag.NewFlagSet("cli", flag.ExitOnError)

	// create a new subcommand
	fs.String("subcommand", "default", "subcommand to execute")

	// parse the command line arguments
	fs.Parse(os.Args[1:])
	fmt.Println("subcommand:", fs.Lookup("subcommand").Value)
}

// Run the application with the following command
// go run cli-interface.go -subcommand=hello
