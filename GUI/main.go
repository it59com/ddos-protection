package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Server Monitor")

	statusLabel := widget.NewLabel("Connecting to server...")
	agentList := widget.NewList(
		func() int {
			return 10 // количество агентов (пример)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Agent Info")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(fmt.Sprintf("Agent %d: Active", i+1)) // пример данных
		},
	)

	myWindow.SetContent(container.NewVBox(
		statusLabel,
		agentList,
	))

	go func() {
		for {
			time.Sleep(5 * time.Second)
			statusLabel.SetText("Server is running...") // здесь можно обновлять состояние сервера
		}
	}()

	myWindow.ShowAndRun()
}
