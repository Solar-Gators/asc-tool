package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/joho/godotenv"
	"github.com/tkrajina/gpxgo/gpx"
)

// physics sim should be main program
func main() {
	err := godotenv.Load() // Load environment variables
	if err != nil {
		fmt.Println(err)
		return
	}

	// fmt.Println(CallApi("maps"))

	file, err := os.Open("ASC-2022-Reference-Route-V2.gpx")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	// Get the file size
	stat, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Read the file into a byte slice
	bs := make([]byte, stat.Size())
	_, err = bufio.NewReader(file).Read(bs)
	if err != nil && err != io.EOF {
		fmt.Println(err)
		return
	}

	gpxFile, err := gpx.ParseBytes(bs)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(len(gpxFile.Routes))

	for _, route := range gpxFile.Routes {
		fmt.Println(route.Name)
		fmt.Println(len(route.Points))
		for _, point := range route.Points {
			if point.Name != "" {
				fmt.Println(point.Name)
			}
		}
		fmt.Println()
	}
}
