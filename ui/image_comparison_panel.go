package ui

import (
	"fmt"
	"image"
	"imgcomp/ui/custom"
	"imgcomp/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ImageComparisonPanel struct {
	container *fyne.Container

	image1Canvas *custom.ClickableImage
	image1Label  *widget.RichText
	image2Canvas *custom.ClickableImage
	image2Label  *widget.RichText
}

func (p *ImageComparisonPanel) Image1Container() fyne.CanvasObject {
	return p.image1Canvas
}

func (p *ImageComparisonPanel) Image2Container() fyne.CanvasObject {
	return p.image2Canvas
}

func (p *ImageComparisonPanel) SetImage(imageNumber int, img *image.Image, path string, fileSize int64) {
	formattedString := fmt.Sprintf(
		"%s\n\n%dx%d | %s bytes",
		path,
		(*img).Bounds().Dx(),
		(*img).Bounds().Dy(),
		util.FormatIntWithSpaces(fileSize))

	var chosenLabel *widget.RichText
	switch imageNumber {
	case 1:
		p.image1Canvas.SetImage(*img)
		chosenLabel = p.image1Label
	case 2:
		p.image2Canvas.SetImage(*img)
		chosenLabel = p.image2Label
	}
	chosenLabel.ParseMarkdown(formattedString)
	for i := range chosenLabel.Segments {
		if seg, ok := chosenLabel.Segments[i].(*widget.TextSegment); ok {
			seg.Style.Alignment = fyne.TextAlignCenter
			chosenLabel.Wrapping = fyne.TextWrapBreak
		}
	}
}

func NewImageComparisonPanel(
	onImageClicked func(imageNumber int),
	onImageDeleted func(imageNumber int),
	onImageIgnored func(),
	algo util.ScalingAlgorithm,
	showManagementButtons bool,
) *ImageComparisonPanel {
	panel := &ImageComparisonPanel{}

	panel.image1Label = widget.NewRichTextFromMarkdown("Image 1")
	panel.image1Label.Wrapping = fyne.TextWrap(fyne.TextAlignCenter)
	panel.image2Label = widget.NewRichTextFromMarkdown("Image 2")
	panel.image2Label.Wrapping = fyne.TextWrap(fyne.TextAlignCenter)

	panel.image1Canvas = custom.NewClickableImage(nil, func() {
		onImageClicked(1)
	}, algo)
	panel.image1Canvas.SetImageMinSize(fyne.NewSize(util.ImageMaxWidth, util.ImageMaxHeight))

	panel.image2Canvas = custom.NewClickableImage(nil, func() {
		onImageClicked(2)
	}, algo)
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

	text1VBox := container.NewVBox(
		panel.image1Label,
	)
	text2VBox := container.NewVBox(
		panel.image2Label,
	)

	img1VBox := container.NewVBox(
		img1Container,
	)
	if showManagementButtons {
		img1VBox.Add(widget.NewButton("Delete", func() {
			onImageDeleted(1)
		}))
	}

	img2VBox := container.NewVBox(
		img2Container,
	)
	if showManagementButtons {
		img2VBox.Add(widget.NewButton("Delete", func() {
			onImageDeleted(2)
		}))
	}

	textRow := container.NewGridWithColumns(2,
		text1VBox,
		text2VBox,
	)

	imageRow := container.NewGridWithColumns(2,
		img1VBox,
		img2VBox,
	)

	panel.container = container.NewVBox(textRow, imageRow)

	if showManagementButtons {
		ignoreButton := widget.NewButton("Ignore", onImageIgnored)
		panel.container.Add(ignoreButton)
	}

	return panel
}

func (p *ImageComparisonPanel) GetContainer() *fyne.Container {
	return p.container
}
