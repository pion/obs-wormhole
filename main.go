package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/app"
)

const (
	RouteHome      = "/home"
	RouteSignaling = "/signaling"
	RouteStatus    = "/status"
)

func main() {
	var cfg RouterConfig
	cfg.Route(RouteHome, func(navigator Navigator, ctx interface{}) (Page, error) {
		return NewHomePage(navigator)
	})

	cfg.Route(RouteSignaling, func(navigator Navigator, ctx interface{}) (Page, error) {
		return NewSignalingPage(navigator, ctx.(SignalingPageContext))
	})

	cfg.InitialPath(RouteHome)

	myApp := app.New()
	myWindow := myApp.NewWindow("OBS Wormhole")
	myWindow.Resize(fyne.Size{Width: 1280, Height: 1024})
	router, err := cfg.Build()
	panicIfErr(err)

	myWindow.SetContent(router)
	myWindow.ShowAndRun()
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
