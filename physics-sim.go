package main

import (
	"fmt"

	"github.com/joho/godotenv"
)

// physics sim should be main program
func main() {
	err := godotenv.Load() // Load environment variables
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(CallApi("maps"))
}
