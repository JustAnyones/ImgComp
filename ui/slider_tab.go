package ui

import (
	"fmt"
	"image"
	"imgcomp/util"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type LayerSliderTab struct {
	imageContainer *fyne.Container
	container      *fyne.Container
	img1           *image.Image
	img2           *image.Image
	scalingAlgo    util.ScalingAlgorithm
}

func (s *LayerSliderTab) RemoveAll() *fyne.Container {
	s.imageContainer.RemoveAll()
	return s.container
}

func (s *LayerSliderTab) Refresh() {
	s.imageContainer.Refresh()
	s.container.Refresh()
}

func (s *LayerSliderTab) Compare(img1, img2 *image.Image, algo util.ScalingAlgorithm) {

	s.img1 = img1
	s.img2 = img2
	s.scalingAlgo = algo

	sizeX, sizeY := util.GetScaledBounds(img1)
	newSize := fyne.NewSize(sizeX, sizeY)

	resized1 := util.RescaleImageFast(*img1, algo)
	resized2 := util.RescaleImageFast(*img2, algo)

	comp1 := canvas.NewImageFromImage(resized1)
	comp1.FillMode = canvas.ImageFillOriginal
	if algo == util.NearestNeighbor {
		comp1.ScaleMode = canvas.ImageScalePixels
	} else {
		comp1.ScaleMode = canvas.ImageScaleFastest
	}
	comp1.SetMinSize(newSize)
	comp1.Resize(newSize)
	comp1.Move(fyne.NewPos(0, 0))

	cropped := util.CropImageFast(&resized2, 0.5, algo)
	comp2 := canvas.NewImageFromImage(cropped)
	comp2.FillMode = canvas.ImageFillOriginal
	if algo == util.NearestNeighbor {
		comp2.ScaleMode = canvas.ImageScalePixels
	} else {
		comp2.ScaleMode = canvas.ImageScaleFastest
	}
	comp2.SetMinSize(newSize)
	comp2.Resize(newSize)
	comp2.Move(fyne.NewPos(0, 0))

	s.imageContainer.Add(comp1)
	s.imageContainer.Add(comp2)
	s.Refresh()
}

func NewLayerSliderTab(algo util.ScalingAlgorithm) *LayerSliderTab {
	sliderSection := &LayerSliderTab{
		imageContainer: container.NewWithoutLayout(),
		scalingAlgo:    algo,
	}

	var debounceTimer *time.Timer
	var debounceMutex sync.Mutex
	slider := widget.NewSlider(0, 1)
	slider.Step = 0.01
	slider.Value = 0.5 // start in the middle
	slider.OnChanged = func(val float64) {
		if len(sliderSection.imageContainer.Objects) < 2 {
			fmt.Println("Slider image container does not have enough objects to update")
			return
		}

		debounceMutex.Lock()
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(10*time.Millisecond, func() {

			_, ok := sliderSection.imageContainer.Objects[1].(*canvas.Image)
			if !ok {
				fmt.Println("Slider image container does not contain a canvas.Image at index 1")
				return
			}
			cropped := util.CropImageFast(sliderSection.img2, val, sliderSection.scalingAlgo)

			sliderSection.imageContainer.Objects[1].(*canvas.Image).Image = cropped
			fyne.Do(func() {
				canvas.Refresh(sliderSection.imageContainer.Objects[1])
			})

			//sliderImageContainer.Objects[1].(*canvas.Image).Refresh()
		})
		debounceMutex.Unlock()
	}
	sliderSection2 := container.NewVBox(
		widget.NewLabel("Slider Section"),
		sliderSection.imageContainer,
		slider,
	)

	sliderSection.container = sliderSection2

	return sliderSection
}

func (s *LayerSliderTab) GetContainer() *fyne.Container {
	return s.container
}
