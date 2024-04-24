/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"regexp"
	"strconv"

	"asc-simulation/phys"

	"github.com/spf13/cobra"
)

// calcCmd represents the calc command
var calcCmd = &cobra.Command{
	Use:   "calc",
	Short: "Calculates battery % and speed",
	Long: `Calculates battery % and speed

    Takes as input:
    - Name of route segment
    - Initial battery %
    - Max Target Speed (mph)
    - Loop Name
    - Loop count
    - Start Time (HH:MM)
    - Checkpoint 1 close time (HH:MM)
    - Checkpoint 2 close time (HH:MM)
    - Checkpoint 3 close time (HH:MM)
    - Stage finish close time (HH:MM)`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 10 {
			panic("Provided too few commands: " + strconv.Itoa(len(args)) + "/10")
		}

		routeSeg := args[0]
		battery, err := strconv.Atoi(args[1])
		if err != nil {
			panic("Battery % must be an integer, not: '" + args[1] + "'")
		}
		targSpeed, err := strconv.Atoi(args[2])
		if err != nil {
			panic("Target Speed must be an integer, not: '" + args[2] + "'")
		}
		loopName := args[3]

		loopCount, err := strconv.Atoi(args[4])
		if err != nil {
			panic("Loop 2 must be an integer, not: '" + args[4] + "'")
		}
		startTime := args[5]
		cpOneClose := args[6]
		cpTwoClose := args[7]
		cpThreeClose := args[8]
		stageClose := args[9]

		for i := 5; i < 10; i++ {
			timeArg := args[i]

			if !regexp.MustCompile(`\d{2}\:\d{2}`).MatchString(timeArg) {
				panic("Argument #" + strconv.Itoa(i) + " not in HH:MM format")
			}
		}
		fmt.Println("Calculating...")
		phys.CalcPhysics(routeSeg, battery, targSpeed, loopName, loopCount, startTime, cpOneClose, cpTwoClose, cpThreeClose, stageClose)
	},
}

func init() {
	rootCmd.AddCommand(calcCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// calcCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// calcCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
