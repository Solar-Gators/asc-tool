//go:build main2

package main

import (
	"bufio"
	"fmt"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
)

var routePath string = "./asc-routes-2024/"
var routeFileType string = ".route.json"

func captureOutput(r io.ReadCloser, l *widget.Label) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		l.SetText(line)
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading from pipe: %v\n", err)
	}
}

func generateRouteList() ([]string, []string) {
	var routeNames []string
	var loopNames []string

	files, err := ioutil.ReadDir(routePath)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return routeNames, loopNames
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), routeFileType) {
			if file.Name()[1] == 'L' {
				loopName := strings.TrimSuffix(file.Name(), routeFileType)
				loopNames = append(loopNames, loopName)
			} else {
				routeName := strings.TrimSuffix(file.Name(), routeFileType)
				routeNames = append(routeNames, routeName)
			}
		}
	}

	return routeNames, loopNames
}

func main() {

	a := app.New()
	w := a.NewWindow("ASC Sim")

	label_1 := widget.NewLabel("Route Segment:")
	routeNames, loopNames := generateRouteList()

	route_segment := widget.NewSelect(routeNames, func(value string) {
	})
	route_segment.SetSelectedIndex(0)

	label_2 := widget.NewLabel("Starting Battery (%):")
	starting_battery := widget.NewEntry()
	starting_battery.SetText("100")

	to_optimize := widget.NewCheck("Optimize speed", func(value bool) {})

	label_3 := widget.NewLabel("Max Speed (mph):")
	max_speed_mph := widget.NewEntry()
	max_speed_mph.SetText("55")

	label_4 := widget.NewLabel("Loop Name:")
	loop_name := widget.NewSelect(loopNames, func(value string) {
	})
	loop_name.SetSelectedIndex(0)

	label_5 := widget.NewLabel("Loop Count:")
	loop_count := widget.NewEntry()
	loop_count.SetText("1")

	label_6 := widget.NewLabel("Checkpoint 1 Close Time (HH:MM):")
	checkpoint_1_time := widget.NewEntry()
	checkpoint_1_time.SetText("10:40")

	label_7 := widget.NewLabel("Checkpoint 2 Close Time (HH:MM):")
	checkpoint_2_time := widget.NewEntry()
	checkpoint_2_time.SetText("13:15")

	label_8 := widget.NewLabel("Checkpoint 3 Close Time (HH:MM):")
	checkpoint_3_time := widget.NewEntry()
	checkpoint_3_time.SetText("16:20")

	label_9 := widget.NewLabel("Stage Close Time (HH:MM):")
	stage_finish_time := widget.NewEntry()
	stage_finish_time.SetText("18:30")

	label_10 := widget.NewLabel("Start Time (HH:MM):")
	start_time := widget.NewEntry()
	start_time.SetText("08:00")

	output_label := widget.NewLabel("Calculating...")
	output_label.Hide()
	output_label.TextStyle.Monospace = true

	output_text := widget.NewLabel("")
	output_text.Hide()
	output_text.TextStyle.Monospace = true

	image1 := canvas.NewImageFromFile("./plots/battery.png")
	image1.FillMode = canvas.ImageFillOriginal

	image2 := canvas.NewImageFromFile("./plots/velocity.png")
	image2.FillMode = canvas.ImageFillOriginal

	image3 := canvas.NewImageFromFile("./plots/energyUsed.png")
	image3.FillMode = canvas.ImageFillOriginal

	image4 := canvas.NewImageFromFile("./plots/energyGained.png")
	image4.FillMode = canvas.ImageFillOriginal

	label_i_1 := widget.NewLabel("Battery")
	label_i_2 := widget.NewLabel("Velocity")
	label_i_3 := widget.NewLabel("Energy Used")
	label_i_4 := widget.NewLabel("Energy Gained")

	image1.Hide()
	image3.Hide()
	image2.Hide()
	image4.Hide()

	label_i_1.Hide()
	label_i_2.Hide()
	label_i_3.Hide()
	label_i_4.Hide()

	go_button := widget.NewButton("Go", func() {
		to_run := "./main.exe"
		to_run_2 := "calc"
		if to_optimize.Checked {
			to_run = "./mystic_venv/bin/python"
			to_run_2 = "./optimizer.py"
		}

		cmd := exec.Command(to_run, to_run_2, routePath+route_segment.Selected+routeFileType, starting_battery.Text, max_speed_mph.Text, routePath+loop_name.Selected+routeFileType, loop_count.Text, start_time.Text, checkpoint_1_time.Text, checkpoint_2_time.Text, checkpoint_3_time.Text, stage_finish_time.Text)

		image1.Hide()
		image3.Hide()
		image2.Hide()
		image4.Hide()

		label_i_1.Hide()
		label_i_2.Hide()
		label_i_3.Hide()
		label_i_4.Hide()

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Printf("Error obtaining stdout: %s\n", err)
			return
		}

		// Getting the pipe for standard error
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			fmt.Printf("Error obtaining stderr: %s\n", err)
			return
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			fmt.Printf("Error starting command: %s\n", err)
			return
		}

		go captureOutput(stdoutPipe, output_text)
		go captureOutput(stderrPipe, output_text)

		output_label.Show()
		output_text.Show()

		err = cmd.Wait()

		image1.Refresh()
		image2.Refresh()
		image3.Refresh()
		image4.Refresh()

		image1.Show()
		image2.Show()
		image3.Show()
		image4.Show()

		label_i_1.Show()
		label_i_2.Show()
		label_i_3.Show()
		label_i_4.Show()

		if err != nil {
			fmt.Printf("Command finished with error: %v\n", err)
		}
	})

	w.SetContent(
		container.NewVScroll(
			container.NewVBox(
				label_1,
				route_segment,
				label_2,
				starting_battery,
				to_optimize,
				label_3,
				max_speed_mph,
				label_10,
				start_time,

				container.NewHSplit(
					container.NewHBox(
						label_4,
						loop_name,
					),

					container.NewHBox(
						label_5,
						loop_count,
					)),

				container.NewHBox(
					label_6,
					label_7,
				),

				container.NewHSplit(
					checkpoint_1_time,
					checkpoint_2_time,
				),

				container.NewHBox(
					label_8,
					label_9,
				),

				container.NewHSplit(
					checkpoint_3_time,
					stage_finish_time,
				),
				go_button,
				output_label,
				output_text,
				container.NewVBox(
					container.NewHBox(
						container.NewVBox(
							label_i_1,
							image1,
						),
						container.NewVBox(
							label_i_2,
							image2,
						),
					),
					container.NewHBox(
						container.NewVBox(
							label_i_3,
							image3,
						),
						container.NewVBox(
							label_i_4,
							image4,
						),
					),
				),
			),
		))

	w.ShowAndRun()
}
