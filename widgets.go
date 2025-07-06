package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// ClickableImage is a custom widget that displays an image and responds to taps.
type ClickableImage struct {
	widget.BaseWidget
	image    *canvas.Image
	onTapped func()
	minSize  fyne.Size
}

// NewClickableImage creates a new ClickableImage widget.
func NewClickableImage(res fyne.Resource, tapped func()) *ClickableImage {
	img := &ClickableImage{
		image:    canvas.NewImageFromResource(res),
		onTapped: tapped,
		minSize:  fyne.NewSize(0, 0),
	}
	img.image.FillMode = canvas.ImageFillOriginal  // Default fill mode
	img.image.ScaleMode = canvas.ImageScaleFastest // Use fastest scaling for performance
	img.ExtendBaseWidget(img)
	return img
}

// SetImageMinSize allows setting the minimum size for the image within the widget.
// Call this after creating the widget if you want to override its default MinSize.
func (c *ClickableImage) SetImageMinSize(size fyne.Size) {
	c.minSize = size
	c.Refresh()
}

// CreateRenderer is a Fyne internal method to create a renderer for the widget.
func (c *ClickableImage) CreateRenderer() fyne.WidgetRenderer {
	return &clickableImageRenderer{
		img:    c.image,
		widget: c,
	}
}

// Tapped is part of the fyne.Tappable interface.
func (c *ClickableImage) Tapped(e *fyne.PointEvent) {
	if c.onTapped != nil {
		c.onTapped()
	}
	//fmt.Printf("Image tapped at: %.1f, %.1f\n", e.Position.X, e.Position.Y)
}

// TappedSecondary is part of the fyne.Tappable interface (e.g., right-click).
func (c *ClickableImage) TappedSecondary(e *fyne.PointEvent) {
	//fmt.Println("Image secondary tapped (right-clicked)")
}

// clickableImageRenderer handles the rendering of the ClickableImage.
type clickableImageRenderer struct {
	img    *canvas.Image
	widget *ClickableImage
}

// MinSize returns the minimum size required by the widget.
func (r *clickableImageRenderer) MinSize() fyne.Size {
	// If a specific minSize is set on the widget, use that.
	// Otherwise, fall back to the image's inherent minimum size.
	if r.widget.minSize.Width > 0 || r.widget.minSize.Height > 0 {
		return r.widget.minSize
	}
	return r.img.MinSize()
}

// Layout sets the position and size of the contained objects.
func (r *clickableImageRenderer) Layout(size fyne.Size) {
	r.img.Resize(size)
}

// Refresh triggers a redraw of the widget.
func (r *clickableImageRenderer) Refresh() {
	r.img.Refresh()
	// Ensure the entire widget is refreshed
	canvas.Refresh(r.widget)
}

// Objects returns the list of canvas objects that make up the widget.
func (r *clickableImageRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.img}
}

// Destroy is called when the renderer is no longer needed.
func (r *clickableImageRenderer) Destroy() {}
