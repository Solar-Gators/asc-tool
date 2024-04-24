package main

import (
	"fmt"

	"asc-simulation/cmd"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load() // Load environment variables
	if err != nil {
		fmt.Println(err)
		return
	}
	cmd.Execute()
}
