package ui

import (
	"image"
	"imgcomp/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type PixelWiseTab struct {
	resultLabel *widget.Label
	diffCanvas  *canvas.Image
	container   *fyne.Container

	showMonochrome     bool
	onMonochromeChange func(bool)
}

func NewPixelWiseTab(algo util.ScalingAlgorithm, onMonochromeChange func(bool)) *PixelWiseTab {
	p := &PixelWiseTab{}
	p.resultLabel = widget.NewLabel("???")
	p.showMonochrome = false
	p.onMonochromeChange = onMonochromeChange

	p.diffCanvas = canvas.NewImageFromImage(nil)
	if algo == util.NearestNeighbor {
		p.diffCanvas.ScaleMode = canvas.ImageScalePixels
	} else {
		p.diffCanvas.ScaleMode = canvas.ImageScaleFastest
	}
	p.diffCanvas.SetMinSize(fyne.NewSize(util.ImageMaxWidth, util.ImageMaxHeight))
	p.diffCanvas.FillMode = canvas.ImageFillContain

	button := widget.NewButton("Toggle Monochrome Diff", func() {
		p.showMonochrome = !p.showMonochrome
		if p.onMonochromeChange != nil {
			p.onMonochromeChange(p.showMonochrome)
		}
	})

	p.container = container.NewVBox(
		p.resultLabel, p.diffCanvas, button,
	)
	return p
}

func (p *PixelWiseTab) SetImage(img *image.Image) {
	p.diffCanvas.Image = *img
	p.diffCanvas.Refresh()
}

func (p *PixelWiseTab) ShowMonochrome() bool {
	return p.showMonochrome
}

func (p *PixelWiseTab) SetMessage(message string) {
	p.resultLabel.SetText(message)
}

func (p *PixelWiseTab) GetContainer() *fyne.Container {
	return p.container
}
