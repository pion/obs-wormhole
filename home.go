package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/widget"
)

type HomePage struct {
	fyne.Widget
}

func NewHomePage(navigator Navigator) (Page, error) {
	errLabel := widget.NewLabel("")
	wormholeType := widget.NewSelect([]string{"WebRTC Source", "P2P Streaming"}, nil)
	webrtcRole := widget.NewSelect([]string{"Offer", "Answer"}, nil)

	form := widget.NewForm(
		widget.NewFormItem("Wormhole Type", wormholeType),
		widget.NewFormItem("WebRTC Role", webrtcRole),
	)

	loading := widget.NewProgressBarInfinite()
	loading.Hide()

	content := widget.NewScrollContainer(
		widget.NewVBox(
			form,
			errLabel,
			loading,
		))

	form.OnSubmit = func() {
		if wormholeType.Selected == "" || webrtcRole.Selected == "" {
			errLabel.SetText("Type and Role must be set")
		} else {
			loading.Show()
			err := navigator.Push(RouteSignaling, SignalingPageContext{
				IsOffer: webrtcRole.Selected == "Offer",
			})
			loading.Hide()
			if err != nil {
				errLabel.SetText(err.Error())
			}
		}
	}

	return &HomePage{content}, nil
}

func (page *HomePage) BeforeDestroy() {

}
