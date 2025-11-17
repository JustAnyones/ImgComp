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
}

func NewPixelWiseTab() *PixelWiseTab {
	p := &PixelWiseTab{}
	p.resultLabel = widget.NewLabel("???")

	p.diffCanvas = canvas.NewImageFromImage(nil)
	p.diffCanvas.SetMinSize(fyne.NewSize(util.ImageMaxWidth, util.ImageMaxHeight))
	p.diffCanvas.FillMode = canvas.ImageFillContain

	p.container = container.NewVBox(
		p.resultLabel, p.diffCanvas,
	)
	return p
}

func (p *PixelWiseTab) SetImage(img *image.Image) {
	p.diffCanvas.Image = *img
	p.diffCanvas.Refresh()
}

func (p *PixelWiseTab) SetMessage(message string) {
	p.resultLabel.SetText(message)
}

func (p *PixelWiseTab) GetContainer() *fyne.Container {
	return p.container
}
