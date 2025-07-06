package main

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var resultLabel *widget.Label

const (
	ImageMaxWidth  = 400 // Maximum width for the images
	ImageMaxHeight = 400 // Maximum height for the images
)

func constructComparison(image1 *image.Image, image2 *image.Image) (*canvas.Image, *canvas.Image) {
	sizeX, sizeY := getScaledBounds(image1)
	newSize := fyne.NewSize(sizeX, sizeY)

	resized1 := rescaleImageFast(*image1)
	resized2 := rescaleImageFast(*image2)

	comp1 := canvas.NewImageFromImage(resized1)
	comp1.FillMode = canvas.ImageFillOriginal
	comp1.ScaleMode = canvas.ImageScaleFastest
	comp1.SetMinSize(newSize)
	comp1.Resize(newSize)
	comp1.Move(fyne.NewPos(0, 0))

	cropped := cropImageFast(&resized2, 0.5)
	comp2 := canvas.NewImageFromImage(cropped)
	comp2.FillMode = canvas.ImageFillOriginal
	comp1.ScaleMode = canvas.ImageScaleFastest
	comp2.SetMinSize(newSize)
	comp2.Resize(newSize)
	comp2.Move(fyne.NewPos(0, 0))
	return comp1, comp2
}

var sliderSection *fyne.Container

var sliderImageContainer *fyne.Container = container.NewWithoutLayout()

var image1Path string
var image2Path string

var imageLabel1 *widget.Label = widget.NewLabel("Image 1")
var imageLabel2 *widget.Label = widget.NewLabel("Image 2")

var image1 *image.Image
var image2 *image.Image

var image1Canvas *ClickableImage // Canvas to display the first image
var image2Canvas *ClickableImage // Canvas to display the second image
var diffCanvas *canvas.Image     // Canvas to display the difference image

// Reference to the main window, used for displaying dialogs and other UI elements.
var mainWindow fyne.Window

// WaitGroup to synchronize loading of images
var loadingWaitGroup = &sync.WaitGroup{}

func renderComparison() {
	startTime := time.Now()
	diff, mae, pixelCount := computeImageDiffFast(image1, image2)
	fmt.Printf("Image difference computed in %v\n", time.Since(startTime))
	diffCanvas.Image = diff
	diffCanvas.Refresh()

	// Update the comparison section with the new images
	sliderImageContainer.RemoveAll()
	if mae > 0 {
		c1, c2 := constructComparison(image1, image2)
		sliderImageContainer.Add(c1)
		sliderImageContainer.Add(c2)
	}

	sliderImageContainer.Refresh()
	sliderSection.Refresh()

	if mae == 0 {
		resultLabel.SetText("Images are identical")
	} else {
		resultLabel.SetText(fmt.Sprintf("Images differ with MAE: %.2f (%d px)", mae, pixelCount))
	}
}

func loadAndRenderImage(path string, index int, wg bool) {
	// Load the image from the specified path
	img, err := loadImage(path)
	if err != nil {
		fmt.Println("Error loading image:", err)
		if wg {
			loadingWaitGroup.Done()
		}
		dialog.ShowError(err, mainWindow) // Show error dialog if image loading fails
		return
	}

	// Determine the file info to get the size and other details
	fileInfo, err := os.Stat(path)
	if err != nil {
		if wg {
			loadingWaitGroup.Done()
		}
		dialog.ShowError(err, mainWindow)
		return
	}

	// Determine which image to update based on the index
	const maxLength = 64
	rescaledImg := rescaleImageFast(img)
	if index == 0 {
		image1Path = path
		image1 = &rescaledImg
		fyne.Do(func() {
			imageLabel1.SetText(fmt.Sprintf("%s (%s bytes)", wrapStringIntelligently(path, maxLength), formatIntWithSpaces(fileInfo.Size())))
			(*image1Canvas).image.Image = img
			(*image1Canvas).Refresh()
		})
	} else {
		image2Path = path
		image2 = &rescaledImg
		fyne.Do(func() {
			imageLabel2.SetText(fmt.Sprintf("%s (%s bytes)", wrapStringIntelligently(path, maxLength), formatIntWithSpaces(fileInfo.Size())))
			(*image2Canvas).image.Image = img
			(*image2Canvas).Refresh()
		})

	}

	// If we're running in a wait group context,
	// we need to indicate that we're done loading this image.
	if wg {
		loadingWaitGroup.Done()
	} else {
		// otherwise, we can directly render the comparison
		// as we're running in the main thread and have both images loaded.
		renderComparison()
	}
}

func main() {
	app := app.New()
	mainWindow = app.NewWindow("Image comparison tool")

	// Canvas elements to display images
	image1Canvas = NewClickableImage(nil, func() {
		if image1Path == "" {
			return
		}
		err := exec.Command("xdg-open", image1Path).Start()
		if err != nil {
			dialog.ShowError(err, mainWindow)
			return
		}
	})
	image1Canvas.SetImageMinSize(fyne.NewSize(ImageMaxWidth, ImageMaxHeight))
	image2Canvas = NewClickableImage(nil, func() {
		if image2Path == "" {
			return
		}
		err := exec.Command("xdg-open", image2Path).Start()
		if err != nil {
			dialog.ShowError(err, mainWindow)
			return
		}
	})
	image2Canvas.SetImageMinSize(fyne.NewSize(ImageMaxWidth, ImageMaxHeight))

	// Create element to display calculated differences.
	diffCanvas = canvas.NewImageFromImage(nil)
	diffCanvas.SetMinSize(fyne.NewSize(ImageMaxWidth, ImageMaxHeight))
	diffCanvas.FillMode = canvas.ImageFillContain

	// Stack the canvas and the clickable button to make the canvas area interactive.
	img1Container := container.NewStack(
		container.NewCenter(widget.NewLabel("Drag an image to view it")),
		image1Canvas,
	)
	img2Container := container.NewStack(
		container.NewCenter(widget.NewLabel("Drag an image to view it")),
		image2Canvas,
	)

	// Set up drag and drop functionality for the window.
	mainWindow.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {

		img1AbsPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(img1Container)
		img2AbsPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(img2Container)

		// check if the drop position is within the bounds of either image container
		isFirstImageDropped := pos.X >= img1AbsPos.X && pos.X <= img1AbsPos.X+img1Container.Size().Width &&
			pos.Y >= img1AbsPos.Y && pos.Y <= img1AbsPos.Y+img1Container.Size().Height
		isSecondImageDropped := pos.X >= img2AbsPos.X && pos.X <= img2AbsPos.X+img2Container.Size().Width &&
			pos.Y >= img2AbsPos.Y && pos.Y <= img2AbsPos.Y+img2Container.Size().Height
		if !isFirstImageDropped && !isSecondImageDropped {
			return
		}

		if len(uris) == 0 {
			return // No files dropped
		}
		filePath := uris[0].Path() // Process only the first dropped file

		if isFirstImageDropped {
			loadAndRenderImage(filePath, 0, false) // Load and render the first image
		} else {
			loadAndRenderImage(filePath, 1, false) // Load and render the second image
		}
	})

	resultLabel = widget.NewLabel("Computed result")

	ignoreButton := widget.NewButton("Ignore", func() {
		if image1 == nil || image2 == nil {
			dialog.ShowInformation("No images loaded", "Please load both images before ignoring.", mainWindow)
			return
		}
		// Append a line to a file
		file, err := os.OpenFile("ignored_images.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			dialog.ShowError(err, mainWindow)
			return
		}
		defer file.Close()
		_, err = file.WriteString(fmt.Sprintf("%s:%s\n", image1Path, image2Path))
		if err != nil {
			dialog.ShowError(err, mainWindow)
			return
		}
		mainWindow.Close()
	})

	topImageRow := container.NewGridWithColumns(2,
		container.NewVBox(
			container.New(layout.NewCenterLayout(), imageLabel1),
			img1Container,
			widget.NewButton("Delete", func() {
				if image1 == nil {
					return
				}

				if image1Path != "" {
					err := os.Remove(image1Path)
					if err != nil {
						dialog.ShowError(err, mainWindow)
						return
					}
					mainWindow.Close()
				}
			}),
		),
		container.NewVBox(
			container.New(layout.NewCenterLayout(), imageLabel2),
			img2Container,
			widget.NewButton("Delete", func() {
				if image2 == nil {
					return
				}

				if image2Path != "" {
					err := os.Remove(image2Path)
					if err != nil {
						dialog.ShowError(err, mainWindow)
						return
					}
					mainWindow.Close()
				}
			}),
		),
	)
	topImageRow2 := container.NewVBox(
		topImageRow,
		ignoreButton,
	)

	// Section for the difference image and result label, using NewMax for the diffCanvas to expand
	diffSection := container.NewVBox(
		resultLabel, diffCanvas,
	)

	_ = NewClickableImage(theme.InfoIcon(), func() {
		fmt.Println("Custom Clickable Image tapped!")
		// show a dialog with image
		dialog.ShowInformation("Image Clicked", "You clicked the custom image!", mainWindow)
	})

	var debounceTimer *time.Timer
	var debounceMutex sync.Mutex
	slider := widget.NewSlider(0, 1)
	slider.Step = 0.01
	slider.Value = 0.5 // start in the middle
	slider.OnChanged = func(val float64) {
		if len(sliderImageContainer.Objects) < 2 {
			fmt.Println("Slider image container does not have enough objects to update")
			return
		}

		debounceMutex.Lock()
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
		debounceTimer = time.AfterFunc(10*time.Millisecond, func() {
			cropped := cropImageFast(image2, val)
			_, ok := sliderImageContainer.Objects[1].(*canvas.Image)
			if !ok {
				fmt.Println("Slider image container does not contain a canvas.Image at index 1")
				return
			}

			sliderImageContainer.Objects[1].(*canvas.Image).Image = cropped
			fyne.Do(func() {
				canvas.Refresh(sliderImageContainer.Objects[1])
			})

			//sliderImageContainer.Objects[1].(*canvas.Image).Refresh()
		})
		debounceMutex.Unlock()
	}
	sliderSection = container.NewVBox(
		widget.NewLabel("Slider Section"),
		sliderImageContainer,
		slider,
	)

	// Main content, using GridWithRows to divide the window vertically between the top images and the difference section
	mainContent := container.NewGridWithRows(3,
		topImageRow2,
		diffSection,
		sliderSection,
	)

	// Set the window content using a border layout.
	mainWindow.SetContent(container.NewVScroll(mainContent))
	mainWindow.Resize(fyne.NewSize(1000, 500)) // Set initial window size

	// Take full paths as command line arguments
	if len(os.Args) > 1 {
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run main.go <image1_path> <image2_path>")
			return
		}
		img1Path := os.Args[1]
		img2Path := os.Args[2]
		if _, err := os.Stat(img1Path); os.IsNotExist(err) {
			fmt.Printf("Image 1 file does not exist: %s\n", img1Path)
			return
		}
		if _, err := os.Stat(img2Path); os.IsNotExist(err) {
			fmt.Printf("Image 2 file does not exist: %s\n", img2Path)
			return
		}
		fmt.Printf("Loading images from command line arguments: %s, %s\n", img1Path, img2Path)
		loadingWaitGroup.Add(2)                  // Add two goroutines to the wait group
		go loadAndRenderImage(img1Path, 0, true) // Load and render the first image
		go loadAndRenderImage(img2Path, 1, true) // Load and render the second image
		loadingWaitGroup.Wait()                  // Wait for both goroutines to finish
		mainWindow.Resize(fyne.NewSize(2560, 1440))
		mainWindow.CenterOnScreen()
		renderComparison()
	} else {
		fmt.Println("No command line arguments provided. Please drag and drop images or use the buttons to load images.")
	}

	mainWindow.ShowAndRun()
}
