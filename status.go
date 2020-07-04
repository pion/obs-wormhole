package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/widget"
	"github.com/pion/webrtc/v3"
)

type StatusPageContext struct {
	StatusChan <-chan webrtc.ICEConnectionState
}

type StatusPage struct {
	fyne.Widget
	destroyChan chan<- struct{}
}

func NewStatusPage(navigator Navigator, ctx StatusPageContext) (Page, error) {
	statusLabel := widget.NewLabel("")

	content := widget.NewVBox(
		statusLabel,
	)

	destroyChan := make(chan struct{})
	go func() {
	loop:
		for {
			select {
			case status := <-ctx.StatusChan:
				statusLabel.SetText(status.String())
			case <-destroyChan:
				break loop
			}
		}
	}()

	return &StatusPage{Widget: content, destroyChan: destroyChan}, nil
}

func (page *StatusPage) BeforeDestroy() {
	page.destroyChan <- struct{}{}
}
