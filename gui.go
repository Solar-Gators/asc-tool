//go:build main2

package main

import (
	"fmt"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"os/exec"
)

func main() {

	a := app.New()
	w := a.NewWindow("ASC Sim")

	label_1 := widget.NewLabel("Route Segment:")
	route_segment := widget.NewSelect([]string{"Segment-1", "Segment-2"}, func(value string) {
	})
	route_segment.SetSelected("Segment-1")

	label_2 := widget.NewLabel("Starting Battery (%):")
	starting_battery := widget.NewEntry()
	starting_battery.SetText("100")

	label_3 := widget.NewLabel("Max Speed (mph):")
	max_speed_mph := widget.NewEntry()
	max_speed_mph.SetText("55")

	label_4 := widget.NewLabel("Loop 1 Count:")
	loop_1_count := widget.NewEntry()
	loop_1_count.SetText("1")

	label_5 := widget.NewLabel("Loop 2 Count:")
	loop_2_count := widget.NewEntry()
	loop_2_count.SetText("1")

	label_6 := widget.NewLabel("Checkpoint 1 Close Time (HH:MM):")
	checkpoint_1_time := widget.NewEntry()
	checkpoint_1_time.SetText("10:40")

	label_7 := widget.NewLabel("Checkpoint 2 Close Time (HH:MM):")
	checkpoint_2_time := widget.NewEntry()
	checkpoint_2_time.SetText("13:15")

	label_8 := widget.NewLabel("Checkpoint 3 Close Time (HH:MM):")
	checkpoint_3_time := widget.NewEntry()
	checkpoint_3_time.SetText("16:20")

	label_9 := widget.NewLabel("Checkpoint 3 Close Time (HH:MM):")
	stage_finish_time := widget.NewEntry()
	stage_finish_time.SetText("18:30")

	output_label := widget.NewLabel("Output:")
	output_label.Hide()
	output_log := widget.NewLabel("")
	output_log.Hide()
	output_log.TextStyle.Monospace = true

	go_button := widget.NewButton("Go", func() {
		cmd := exec.Command("./asc-simulation", "calc", route_segment.Selected, starting_battery.Text, max_speed_mph.Text, loop_1_count.Text, loop_2_count.Text, checkpoint_1_time.Text, checkpoint_2_time.Text, checkpoint_3_time.Text, stage_finish_time.Text)
		output, err := cmd.CombinedOutput()

		if err != nil {
			fmt.Printf("Error executing command: %s\n", err)
			return
		}

		output_log.SetText(string(output))
		output_label.Show()
		output_log.Show()
	})

	w.SetContent(container.NewVBox(
		label_1,
		route_segment,
		label_2,
		starting_battery,
		label_3,
		max_speed_mph,

		container.NewHSplit(
			container.NewHBox(
				label_4,
				loop_1_count,
			),

			container.NewHBox(
				label_5,
				loop_2_count,
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
		output_log,
	))

	w.ShowAndRun()
}
