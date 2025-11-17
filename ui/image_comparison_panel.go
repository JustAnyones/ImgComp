package ui

import (
	"image"
	"imgcomp/ui/custom"
	"imgcomp/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type ImageComparisonPanel struct {
	container *fyne.Container

	image1Canvas *custom.ClickableImage
	image1Label  *widget.Label
	image2Canvas *custom.ClickableImage
	image2Label  *widget.Label
}

func (p *ImageComparisonPanel) Image1Container() fyne.CanvasObject {
	return p.image1Canvas
}

func (p *ImageComparisonPanel) Image2Container() fyne.CanvasObject {
	return p.image2Canvas
}

func (p *ImageComparisonPanel) SetImage(imageNumber int, img *image.Image, text string) {
	if imageNumber == 1 {
		p.image1Canvas.SetImage(*img)
		p.image1Label.SetText(text)
	} else if imageNumber == 2 {
		p.image2Canvas.SetImage(*img)
		p.image2Label.SetText(text)
	}
}

func NewImageComparisonPanel(
	onImageClicked func(imageNumber int),
	onImageDeleted func(imageNumber int),
	onImageIgnored func(),
) *ImageComparisonPanel {
	panel := &ImageComparisonPanel{}

	panel.image1Label = widget.NewLabel("Image 1")
	panel.image2Label = widget.NewLabel("Image 2")

	panel.image1Canvas = custom.NewClickableImage(nil, func() {
		onImageClicked(1)
	})
	panel.image1Canvas.SetImageMinSize(fyne.NewSize(util.ImageMaxWidth, util.ImageMaxHeight))

	panel.image2Canvas = custom.NewClickableImage(nil, func() {
		onImageClicked(2)
	})
	panel.image2Canvas.SetImageMinSize(fyne.NewSize(util.ImageMaxWidth, util.ImageMaxHeight))

	// Stack the canvas and the clickable button to make the canvas area interactive.
	img1Container := container.NewStack(
		container.NewCenter(widget.NewLabel("Drag an image to view it")),
		panel.image1Canvas,
	)
	img2Container := container.NewStack(
		container.NewCenter(widget.NewLabel("Drag an image to view it")),
		panel.image2Canvas,
	)

	imageRow := container.NewGridWithColumns(2,
		container.NewVBox(
			container.New(layout.NewCenterLayout(), panel.image1Label),
			img1Container,
			widget.NewButton("Delete", func() {
				onImageDeleted(1)
			}),
		),
		container.NewVBox(
			container.New(layout.NewCenterLayout(), panel.image2Label),
			img2Container,
			widget.NewButton("Delete", func() {
				onImageDeleted(2)
			}),
		),
	)

	ignoreButton := widget.NewButton("Ignore", onImageIgnored)

	panel.container = container.NewVBox(
		imageRow,
		ignoreButton,
	)

	return panel
}

func (p *ImageComparisonPanel) GetContainer() *fyne.Container {
	return p.container
}
