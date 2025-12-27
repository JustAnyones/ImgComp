package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"os/exec"
	"sync"
	"time"

	"imgcomp/ui"
	"imgcomp/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

var pixelWiseTab *ui.PixelWiseTab
var layerSliderTab *ui.LayerSliderTab

var comparisonPanel *ui.ImageComparisonPanel

var image1Path string
var image2Path string

var image1 *image.Image
var image2 *image.Image

var scalingAlgo util.ScalingAlgorithm

// Reference to the main window, used for displaying dialogs and other UI elements.
var mainWindow fyne.Window

// WaitGroup to synchronize loading of images
var loadingWaitGroup = &sync.WaitGroup{}

func renderComparison() {
	startTime := time.Now()
	diff, mae, pixelCount := util.ComputeImageDiffFast(image1, image2, scalingAlgo, pixelWiseTab.ShowMonochrome())
	fmt.Printf("Image difference computed in %v\n", time.Since(startTime))

	pixelWiseTab.SetImage(&diff)

	// Update the comparison section with the new images
	layerSliderTab.RemoveAll()
	if mae > 0 {
		layerSliderTab.Compare(image1, image2, scalingAlgo)
	}

	layerSliderTab.Refresh()

	if mae == 0 {
		pixelWiseTab.SetMessage("Images are identical")
	} else {
		pixelWiseTab.SetMessage(fmt.Sprintf("Images differ with MAE: %.2f (%d px)", mae, pixelCount))
	}
}

func loadAndRenderImage(path string, index int, wg bool) {
	// Load the image from the specified path
	img, err := util.LoadImage(path)
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
	rescaledImg := util.RescaleImageFast(img, scalingAlgo)
	if index == 0 {
		image1Path = path
		image1 = &rescaledImg
		fyne.Do(func() {
			comparisonPanel.SetImage(1, &img, path, fileInfo.Size())
		})
	} else {
		image2Path = path
		image2 = &rescaledImg
		fyne.Do(func() {
			comparisonPanel.SetImage(2, &img, path, fileInfo.Size())
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
	// Define flags for command-line arguments
	image1Flag := flag.String("image1", "", "Path to the first image")
	image2Flag := flag.String("image2", "", "Path to the second image")
	showManagementButtonsFlag := flag.Bool("show-management-buttons", true, "Show image management buttons (delete, ignore)")
	scalingAlgoFlag := flag.String("scaling-algo", "bilinear", "Image scaling algorithm (bilinear, nearest)")
	useTrashFlag := flag.Bool("use-trash", false, "Use system trash for deletions")
	flag.Parse()

	if *useTrashFlag {
		// Check if 'trash' command is available
		_, err := exec.LookPath("trash")
		if err != nil {
			fmt.Println("Error: 'trash' command not found. Please install it or disable the use-trash option.")
			return
		}
	}

	switch *scalingAlgoFlag {
	case "nearest":
		scalingAlgo = util.NearestNeighbor
	default:
		scalingAlgo = util.Bilinear
	}

	app := app.New()
	mainWindow = app.NewWindow("Image comparison tool")

	comparisonPanel = ui.NewImageComparisonPanel(
		// onImageClicked
		func(imageNumber int) {
			path := image1Path
			if imageNumber == 2 {
				path = image2Path
			}

			err := exec.Command("xdg-open", path).Start()
			if err != nil {
				dialog.ShowError(err, mainWindow)
				return
			}
		},
		// onImageDeleted
		func(imageNumber int) {
			image := image1
			path := image1Path
			if imageNumber == 2 {
				image = image2
				path = image2Path
			}

			if image == nil {
				return
			}

			if path != "" {
				var err error
				if *useTrashFlag {
					err = util.MoveFileToTrash(path)
				} else {
					err = os.Remove(path)
				}
				if err != nil {
					dialog.ShowError(err, mainWindow)
					return
				}
				mainWindow.Close()
			}
		},
		// onImageIgnored
		func() {
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
		},
		scalingAlgo,
		*showManagementButtonsFlag,
	)

	// Set up drag and drop functionality for the window.
	mainWindow.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {

		img1Container := comparisonPanel.Image1Container()
		img2Container := comparisonPanel.Image2Container()

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

	// Create tabs for Difference and Slider sections
	pixelWiseTab = ui.NewPixelWiseTab(scalingAlgo, func(monochrome bool) {
		if image1 != nil && image2 != nil {
			renderComparison()
		}
	})
	layerSliderTab = ui.NewLayerSliderTab(scalingAlgo)

	tabs := container.NewAppTabs(
		container.NewTabItem("Difference", pixelWiseTab.GetContainer()),
		container.NewTabItem("Layer Slider", layerSliderTab.GetContainer()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	mainContent := container.NewVBox(
		comparisonPanel.GetContainer(),
		tabs,
	)

	// Set the window content using a border layout.
	mainWindow.SetContent(container.NewVScroll(mainContent))
	mainWindow.Resize(fyne.NewSize(1000, 500)) // Set initial window size

	// Load images if provided via flags
	if *image1Flag != "" && *image2Flag != "" {
		img1Path := *image1Flag
		img2Path := *image2Flag
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
