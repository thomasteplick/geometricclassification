/*
Geometric Classification displays the internal structure of 3D geometric objects
such as ellipsoids, parabloids, cubes, boxes, planes, or cones.  It slices the
geometric objects along axial planes in the Cartesian coordinate system.
The object can be solids as well as surfaces.
It gives an overview of the planes in i, j, k axes along with the option
of zooming in on a particular axial plane.  It is possible to select and
view particular planes in the geometric object with different step sizes.
*/

package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/thomasteplick/geometricObject"
)

const (
	addr                           = "127.0.0.1:8080"                  // http server listen address
	fileGeometricDisplay           = "templates/geometricdisplay.html" // html for geometric object plot
	fileGeometricClassification    = "templates/geometricclassification.html"
	patterngeometricdisplay        = "/geometricdisplay"        // http handler display
	patterngeometricclassification = "/geometricclassification" // http handler classification
	xlabelsZoom                    = 11                         // # labels on x axis in zoom
	ylabelsZoom                    = 11                         // # labels on y axis in zoom
	ylabelsOverview                = 3                          // # labels on y axis in overview
	geometricobject                = "geometricobject.txt"      // 3D geometric object file containing the densities, 50x50x50
	geometricrefdims               = "geometricrefdim.txt"      // dimension of references
	dataDir                        = "data/"                    // directory for geometric objects
	rows                           = 300                        // rows in canvas
	cols                           = 300                        // columns in canvas
	planeDim                       = 50                         // number of cells in a plane in x and y in overview
	naxes                          = 3                          // number of axes in Cartesian coordinates
	nplanes                        = 12                         // number of planes per axis (2 rows) in overview
	nplanes2                       = nplanes / 2                // number of planes in each row in overview
	axisDim                        = 100                        // number of cells in each axis in y direction in overview
	deg2rad                        = math.Pi / 180.0            // convert degrees to radians
	classes                        = 19                         // number of classes or geometric objects
)

// test statistics that are tabulated in HTML
type Results struct {
	Class   string // int
	Correct string // int      percent correct
	Count   string // int      number of training examples in the class
}

// Type to contain all the HTML template actions
type PlotT struct {
	Grid            []string  // plotting grid
	Status          string    // status of the plot
	Xlabel          []string  // x-axis labels
	Ylabel          []string  // y-axis labels
	XlabelContainer string    // x-axis labels
	YlabelContainer string    // x-axis labels
	Domain          string    // Overview or Zoom
	TestResults     []Results // tabulated statistics of testing
	TotalCount      string    // total test samples
	TotalCorrect    string    // total correct classification
}

type PlaneMass struct {
	row [50]int
	col [50]int
}

type PlaneDim struct {
	nrows int
	ncols int
}

// classification results
type Stats struct {
	correct    []int // % correct classifcation
	classCount []int // #samples in each class
}

// Type to hold geometric classifcation and display attributes
type Geometric struct {
	density     [][][]byte // geometric object 3D densities
	plot        *PlotT
	planeStarti int // overview starting plane in i axis
	planeStartj int // overview starting plane in j axis
	planeStartk int // overview starting plane in k axis
	planeStep   int // overiew plane step size
	Endpoints
	density2grayscale [10]string
	rotateAxis        string
	rotateAngle       float64
	geoRefMass        [][]PlaneMass  // [axis][plane]
	geoRefDims        [][][]PlaneDim // [class][axis][plane]
	pcError           []float64      // classification percent error
	nsamples          int            // #samples to classify
	noiseLevel        int            // noise level in the samples
	shift             bool           // shift the geometric object
	statistics        Stats
	fmass             *os.File // file handle for reference masses
}

// Type to hold the minimum and maximum data values
type Endpoints struct {
	xmin float64
	xmax float64
	ymin float64
	ymax float64
}

// channel data
type MassError struct {
	sqerr float64
}

// global variables
var (
	// for parse and execution of the html template,
	tmplGeometricDisplay        *template.Template
	tmplGeometricClassification *template.Template
	// 3D geometric objects that can be created and classified
	geometricObjects = map[int]string{
		0:  "ellipsoidsurface",
		1:  "ellipsoidsolid",
		2:  "plane",
		3:  "paraboloid",
		4:  "paraboloidsolid",
		5:  "cube",
		6:  "cone",
		7:  "conesolid",
		8:  "box",
		9:  "hyperbolicparaboloid",
		10: "cylindersurface",
		11: "cylindersolid",
		12: "potentialwell",
		13: "cardioidrevolution",
		14: "cardioidrevolutionsolid",
		15: "lemniscaterevolution",
		16: "lemniscaterevolutionsolid",
		17: "rose4leafrevolution",
		18: "rose4leafrevolutionsolid",
	}
)

// init parses the html template files
func init() {
	tmplGeometricClassification = template.Must(template.ParseFiles(fileGeometricClassification))
	tmplGeometricDisplay = template.Must(template.ParseFiles(fileGeometricDisplay))
}

// Construct a Geometric instance for classification
func newGeometricClassification(r *http.Request, plot *PlotT, nsamples int) (*Geometric, error) {

	// get noise level
	txt := r.FormValue("noiselevel")
	if len(txt) == 0 {
		fmt.Printf("enter noise level\n")
		return nil, fmt.Errorf("enter noise level")
	}

	noiseLevel, err := strconv.Atoi(txt)
	if err != nil {
		fmt.Printf("noiseLevel int conversion error: %v\n", err)
		return nil, fmt.Errorf("noiseLevel int conversion error: %v", err.Error())
	}

	txt = r.FormValue("shiftgeometric")
	shift := false
	if txt == "shiftgeometric" {
		shift = true
	}

	// allocate memory for containers

	densities := make([][][]byte, planeDim)
	for i := range densities {
		densities[i] = make([][]byte, planeDim)
		for j := range densities[i] {
			densities[i][j] = make([]byte, planeDim)
		}
	}

	planeMass := make([][]PlaneMass, naxes)
	for i := range planeMass {
		planeMass[i] = make([]PlaneMass, planeDim)
	}

	planeDims := make([][][]PlaneDim, classes)
	for i := range planeDims {
		planeDims[i] = make([][]PlaneDim, naxes)
		for j := range planeDims[i] {
			planeDims[i][j] = make([]PlaneDim, planeDim)
		}
	}

	fdim, err := os.Open(geometricrefdims)
	if err != nil {
		fmt.Printf("open %s error: %v\n", geometricrefdims, err.Error())
		return nil, fmt.Errorf("open %s error: %v", geometricrefdims, err.Error())
	}
	defer fdim.Close()

	// read in the geometric reference dimensions in order:
	// [class][axis][plane]planeDim
	// class 0, 1, ..., classes-1
	// axis 0, 1, 2
	// plane0 nrows,ncols
	// ...
	// plane49 nrows,ncols
	nrows := 0
	ncols := 0
	for i := range planeDims {
		for j := range planeDims[i] {
			for k := range planeDims[i][j] {
				fmt.Fscanf(fdim, "%d %d\n", &nrows, &ncols)
				planeDims[i][j][k] = PlaneDim{nrows: nrows, ncols: ncols}
			}
		}
	}

	geo := Geometric{
		plot:       plot,
		density:    densities,
		nsamples:   nsamples,
		noiseLevel: noiseLevel,
		shift:      shift,
		Endpoints: Endpoints{
			ymin: 0.0,
			ymax: 100.0,
			xmin: 0,
			xmax: float64(classes - 1),
		},
		pcError: make([]float64, classes),
		statistics: Stats{
			correct:    make([]int, classes),
			classCount: make([]int, classes)},
		geoRefMass: planeMass,
		geoRefDims: planeDims,
	}

	return &geo, nil
}

// Construct a Geometric instance containing state for Display
func newGeometricDisplay(r *http.Request, plot *PlotT, f *os.File) (*Geometric, error) {

	var (
		planeStep   int
		planeStarti int
		planeStartj int
		planeStartk int
		err         error
	)
	densities := make([][][]byte, planeDim)
	for i := range densities {
		densities[i] = make([][]byte, planeDim)
		for j := range densities[i] {
			densities[i][j] = make([]byte, planeDim)
		}
	}

	// Get the planeStarti, planeStartj, planeStartk, planeStep from html form
	txt := r.FormValue("planestep")
	if len(txt) == 0 {
		// assign defaults to plane parameters
		planeStep = 4
		planeStarti = 0
		planeStartj = 0
		planeStartk = 0

	} else {
		planeStep, err = strconv.Atoi(txt)
		if err != nil {
			fmt.Printf("Atoi for planeStep error: %v\n", err.Error())
			return nil, fmt.Errorf("enter plane step")
		}

		txt = r.FormValue("planestarti")
		planeStarti, err = strconv.Atoi(txt)
		if err != nil {
			fmt.Printf("Atoi for planeStarti error: %v\n", err.Error())
			return nil, fmt.Errorf("enter plane start axis i")
		}

		txt = r.FormValue("planestartj")
		planeStartj, err = strconv.Atoi(txt)
		if err != nil {
			fmt.Printf("Atoi for planeStartj error: %v\n", err.Error())
			return nil, fmt.Errorf("enter plane start axis j")
		}

		txt = r.FormValue("planestartk")
		planeStartk, err = strconv.Atoi(txt)
		if err != nil {
			fmt.Printf("Atoi for planeStartk error: %v\n", err.Error())
			return nil, fmt.Errorf("enter plane start axis k")
		}
	}

	// Read the geometric object file containing the densities
	for i := range planeDim {
		for j := range planeDim {
			for k := range planeDim - 1 {
				_, err := fmt.Fscanf(f, "%d", &densities[i][j][k])
				if err != nil {
					fmt.Printf("Fscanf for densities[%d][%d][%d] error: %v\n", i, j, k, err.Error())
					return nil, fmt.Errorf("function Fscanf for densities[%d][%d][%d] error: %v", i, j, k, err.Error())
				}
			}
			_, err := fmt.Fscanf(f, "%d\n", &densities[i][j][planeDim-1])
			if err != nil {
				fmt.Printf("Fscanf for densities[%d][%d] newline error: %v\n", i, j, err.Error())
				return nil, fmt.Errorf("function Fscanf for densities[%d][%d] newline error: %v", i, j, err.Error())
			}
		}
	}

	geo := Geometric{
		plot:        plot,
		planeStarti: planeStarti,
		planeStartj: planeStartj,
		planeStartk: planeStartk,
		planeStep:   planeStep,
		density:     densities,
		rotateAngle: 0,
		rotateAxis:  "",
	}
	// Create density2grayscale map
	geo.density2grayscale = [10]string{"gs0", "gs1", "gs2", "gs3", "gs4",
		"gs5", "gs6", "gs7", "gs8", "gs9"}

	// Used in zoom, not overview
	geo.Endpoints = Endpoints{
		xmin: 0,
		xmax: planeDim,
		ymin: 0,
		ymax: planeDim,
	}

	return &geo, nil
}

// draw horizontal and vertical lines for Overview
func (geo *Geometric) drawCrossedLines() {
	// draw crossed horizontal and vertical lines for overview
	// loop over 5 horizontal lines, make axis separation double thick
	for y := 0; y < rows; y += planeDim {
		for x := range cols {
			row := y
			col := x
			geo.plot.Grid[row*cols+col] = "online"
		}
	}
	// draw separation between axes double thick
	y := 100
	for x := range cols {
		col := x
		row := y - 1
		geo.plot.Grid[row*cols+col] = "online"
		row = 2*y - 1
		geo.plot.Grid[row*cols+col] = "online"
		row = y + 1
		geo.plot.Grid[row*cols+col] = "online"
		row = 2*y + 1
		geo.plot.Grid[row*cols+col] = "online"
	}

	// loop over 5 vertical
	for x := 0; x < cols; x += planeDim {
		for y = range rows {
			col := x
			row := y
			geo.plot.Grid[row*cols+col] = "online"
		}
	}
}

// gridFillInterp inserts the data points in the grid and draws a straight line between points
func (geo *Geometric) gridFillInterp() error {
	var (
		x            float64 = 0.0
		y            float64
		prevX, prevY float64
		xscale       float64
		yscale       float64
	)

	// Put the percent correct in data
	for pat := range geo.pcError {
		geo.pcError[pat] = float64(geo.statistics.correct[pat]) / float64(geo.statistics.classCount[pat]) * 100.0
	}
	y = geo.pcError[0]

	// Mark the data x-y coordinate online at the corresponding
	// grid row/column.

	// Calculate scale factors for x and y
	xscale = float64(cols-1) / (geo.xmax - geo.xmin)
	yscale = float64(rows-1) / (geo.ymax - geo.ymin)

	geo.plot.Grid = make([]string, rows*cols)

	// This cell location (row,col) is on the line
	row := int((geo.ymax-y)*yscale + .5)
	col := int((x-geo.xmin)*xscale + .5)
	geo.plot.Grid[row*cols+col] = "online"

	prevX = x
	prevY = y

	// Scale factor to determine the number of interpolation points
	lenEPy := geo.ymax - geo.ymin
	lenEPx := geo.xmax - geo.xmin

	// Continue with the rest of the points in the file
	for i := 1; i < len(geo.pcError); i++ {
		x++
		// mse/epoch
		y = geo.pcError[i]

		// This cell location (row,col) is on the line
		row := int((geo.ymax-y)*yscale + .5)
		col := int((x-geo.xmin)*xscale + .5)
		geo.plot.Grid[row*cols+col] = "online"

		// Interpolate the points between previous point and current point

		/* lenEdge := math.Sqrt((x-prevX)*(x-prevX) + (y-prevY)*(y-prevY)) */
		lenEdgeX := math.Abs((x - prevX))
		lenEdgeY := math.Abs(y - prevY)
		ncellsX := int(float64(cols) * lenEdgeX / lenEPx) // number of points to interpolate in x-dim
		ncellsY := int(float64(rows) * lenEdgeY / lenEPy) // number of points to interpolate in y-dim
		// Choose the biggest
		ncells := max(ncellsY, ncellsX)

		stepX := (x - prevX) / float64(ncells)
		stepY := (y - prevY) / float64(ncells)

		// loop to draw the points
		interpX := prevX
		interpY := prevY
		for i := 0; i < ncells; i++ {
			row := int((geo.ymax-interpY)*yscale + .5)
			col := int((interpX-geo.xmin)*xscale + .5)
			geo.plot.Grid[row*cols+col] = "online"
			interpX += stepX
			interpY += stepY
		}

		// Update the previous point with the current point
		prevX = x
		prevY = y
	}
	return nil
}

// insert test results into table for display
func (geo *Geometric) tabulateTestResults() error {
	geo.plot.TestResults = make([]Results, classes)

	totalCount := 0
	totalCorrect := 0
	classCount := 0
	// tabulate TestResults by converting numbers to string in Results
	for i := range geo.plot.TestResults {
		classCount = geo.statistics.classCount[i]
		totalCount += classCount
		totalCorrect += geo.statistics.correct[i]
		if classCount > 0 {
			geo.plot.TestResults[i] = Results{
				Class:   strconv.Itoa(i),
				Count:   strconv.Itoa(classCount),
				Correct: strconv.Itoa(geo.statistics.correct[i] * 100 / classCount),
			}
		} else {
			geo.plot.TestResults[i] = Results{
				Class:   strconv.Itoa(i),
				Count:   strconv.Itoa(classCount),
				Correct: "0",
			}
		}
	}
	geo.plot.TotalCount = strconv.Itoa(totalCount)
	geo.plot.TotalCorrect = strconv.Itoa(totalCorrect * 100 / totalCount)
	return nil
}

// Axial planes overview
func (geo *Geometric) gridFillOverview() error {
	geo.xmin = 0
	geo.xmax = cols
	geo.ymin = 0.0
	geo.ymax = rows

	/**************** axis i *******************/
	planeCnt := 0
	planeStopi := min(planeDim, nplanes*geo.planeStep+geo.planeStarti)

	// loop over the planes
	for i := geo.planeStarti; i < planeStopi; i += geo.planeStep {
		ystart := axisDim - (planeCnt/nplanes2)*planeDim
		xstart := (planeCnt % nplanes2) * planeDim
		for j := 0; j < planeDim; j++ {
			y := ystart - j
			for k := 0; k < planeDim; k++ {
				x := xstart + k
				row := int(geo.ymax - float64(y))
				col := int(float64(x) - geo.xmin)
				geo.plot.Grid[row*cols+col] = geo.density2grayscale[geo.density[i][j][k]]
			}
		}
		planeCnt++
	}

	/************* axis j *******************/
	planeCnt = 0
	planeStopj := min(planeDim, nplanes*geo.planeStep+geo.planeStartj)

	// loop over the planes
	for j := geo.planeStartj; j < planeStopj; j += geo.planeStep {
		ystart := 2*axisDim - (planeCnt/nplanes2)*planeDim
		xstart := (planeCnt % nplanes2) * planeDim
		for i := 0; i < planeDim; i++ {
			y := ystart - i
			for k := 0; k < planeDim; k++ {
				x := xstart + k
				row := int(geo.ymax - float64(y))
				col := int(float64(x) - geo.xmin)
				geo.plot.Grid[row*cols+col] = geo.density2grayscale[geo.density[i][j][k]]
			}
		}
		planeCnt++
	}

	/******************* axis k ***********************/
	planeCnt = 0
	planeStopk := min(planeDim, nplanes*geo.planeStep+geo.planeStartk)
	// loop over the planes
	for k := geo.planeStartk; k < planeStopk; k += geo.planeStep {
		ystart := 3*axisDim - (planeCnt/nplanes2)*planeDim
		xstart := (planeCnt % nplanes2) * planeDim
		for i := 0; i < planeDim; i++ {
			y := ystart - i
			for j := 0; j < planeDim; j++ {
				x := xstart + j
				row := int(geo.ymax - float64(y))
				col := int(float64(x) - geo.xmin)
				geo.plot.Grid[row*cols+col] = geo.density2grayscale[geo.density[i][j][k]]
			}
		}
		planeCnt++
	}

	geo.drawCrossedLines()
	return nil
}

// Zoom plot for the given axis and plane
func (geo *Geometric) gridFillZoom(zoomAxis string, zoomPlane int) error {
	const dupl = 6

	// determine which axis i, j, or k
	switch zoomAxis {
	// axis i
	case "i":
		// convert 50x50 plane to 300x300 grid by duplicating 6x in x and y
		for yin := 0; yin < planeDim; yin++ {
			yout := yin * dupl
			for xin := 0; xin < planeDim; xin++ {
				xout := xin * dupl
				// duplicate the input plane 6x in plot.Grid
				for i := 0; i < dupl; i++ {
					row := yout + i
					for j := 0; j < dupl; j++ {
						col := xout + j
						geo.plot.Grid[row*cols+col] = geo.density2grayscale[geo.density[zoomPlane][yin][xin]]
					}
				}
				xout += dupl
			}
		}

	// axis j
	case "j":
		// convert 50x50 plane to 300x300 grid by duplicating 6x in x and y
		for yin := 0; yin < planeDim; yin++ {
			yout := yin * dupl
			for xin := 0; xin < planeDim; xin++ {
				xout := xin * dupl
				// duplicate the input plane 6x in plot.Grid
				for i := 0; i < dupl; i++ {
					row := yout + i
					for j := 0; j < dupl; j++ {
						col := xout + j
						geo.plot.Grid[row*cols+col] = geo.density2grayscale[geo.density[yin][zoomPlane][xin]]
					}
				}
				xout += dupl
			}
		}

	// axis k
	case "k":
		// convert 50x50 plane to 300x300 grid by duplicating 6x in x and y
		for yin := 0; yin < planeDim; yin++ {
			yout := yin * dupl
			for xin := 0; xin < planeDim; xin++ {
				xout := xin * dupl
				// duplicate the input plane 6x in plot.Grid
				for i := 0; i < dupl; i++ {
					row := yout + i
					for j := 0; j < dupl; j++ {
						col := xout + j
						geo.plot.Grid[row*cols+col] = geo.density2grayscale[geo.density[yin][xin][zoomPlane]]
					}
				}
				xout += dupl
			}
		}
	}

	return nil
}

// insertLabels inserts x- an y-axis labels in the plot
func (geo *Geometric) insertLabels() {
	// Construct x-axis labels
	incr := (geo.xmax - geo.xmin) / (xlabelsZoom - 1)
	x := geo.xmin
	for i := range geo.plot.Xlabel {
		geo.plot.Xlabel[i] = fmt.Sprintf("%.2f", x)
		x += incr
	}

	// Construct the y-axis labels
	incr = (geo.ymax - geo.ymin) / (ylabelsZoom - 1)
	y := geo.ymin
	for i := range geo.plot.Ylabel {
		geo.plot.Ylabel[i] = fmt.Sprintf("%.2f", y)
		y += incr
	}
}

// Expand a particular axial plane in the geometric object
func (geo *Geometric) processZoom(zoomAxis string, zoomPlane int) error {
	geo.plot.Grid = make([]string, rows*cols)
	geo.plot.Xlabel = make([]string, xlabelsZoom)
	geo.plot.Ylabel = make([]string, ylabelsZoom)

	// Put expanded axial plane in PlotT grid
	err := geo.gridFillZoom(zoomAxis, zoomPlane)
	if err != nil {
		return fmt.Errorf("gridFillZoom() error: %v", err)
	}

	// insert x-labels and y-labels in PlotT
	geo.insertLabels()

	// different alignment
	geo.plot.YlabelContainer = "ylabel-zoom"
	geo.plot.XlabelContainer = "xlabel-zoom"

	// plot type
	if geo.rotateAngle == 0 {
		geo.plot.Domain = fmt.Sprintf("Axial Plane Zoom, Axis=%s, Plane=%d", zoomAxis, zoomPlane)
	} else {
		geo.plot.Domain = fmt.Sprintf("Axial Plane Zoom, Axis=%s, Plane=%d, Rotate Axis=%s, Rotate Angle=%.0f",
			zoomAxis, zoomPlane, geo.rotateAxis, geo.rotateAngle)
	}

	return nil
}

// Show sequences of axial planes of the geometric object
func (geo *Geometric) processOverview() error {
	geo.plot.Grid = make([]string, rows*cols)
	geo.plot.Xlabel = make([]string, 2)
	geo.plot.Ylabel = make([]string, 3)

	// Put axial planes in PlotT grid
	err := geo.gridFillOverview()
	if err != nil {
		return fmt.Errorf("gridFillOverview() error: %v", err)
	}

	// Construct the y-axis labels, specify the axes
	geo.plot.Ylabel = make([]string, ylabelsOverview)
	y := []string{"i", "j", "k"}
	for i := range geo.plot.Ylabel {
		geo.plot.Ylabel[i] = y[i]
	}

	// different alignment
	geo.plot.YlabelContainer = "ylabel-overview"
	geo.plot.XlabelContainer = "xlabel-overview"

	// Construct the x-axis labels, just note planes
	geo.plot.Xlabel[0] = "Axial"
	geo.plot.Xlabel[1] = "Planes"

	// plot type
	if geo.rotateAngle == 0 {
		geo.plot.Domain = fmt.Sprintf("Axial Plane Overview, Plane Step = %d, Axis i start = %d, Axis j start = %d, Axis k start = %d",
			geo.planeStep, geo.planeStarti, geo.planeStartj, geo.planeStartk)
	} else {
		geo.plot.Domain = fmt.Sprintf("Axial Plane Overview, Plane Step = %d, Axis i start = %d, Axis j start = %d, Axis k start = %d, "+
			"Rotate Axis=%s, Rotate Angle=%.0f", geo.planeStep, geo.planeStarti, geo.planeStartj, geo.planeStartk, geo.rotateAxis, geo.rotateAngle)
	}

	return nil
}

// Rotate axis planes
func (geo *Geometric) rotatePlanes(axis string, angle float64) error {
	// create temporary storage for rotated plane densities
	densityRotated := make([][]byte, planeDim)
	for i := range densityRotated {
		densityRotated[i] = make([]byte, planeDim)
	}

	// mean of the index
	const u = float64(planeDim/2) - .5

	// determine which axis i, j, or k
	// loop over the planes in axis
	// clear both locations for every plane
	switch axis {
	// axis i
	case "i":
		// loop over the planes
		for plane := 0; plane < planeDim; plane++ {

			// Clear the temporary storage for every plane rotation
			for i := range densityRotated {
				for j := range densityRotated[i] {
					densityRotated[i][j] = 0
				}
			}

			// rotate angle and store in temp storage
			// remove the mean to rotate, put it back after
			for yin := 0; yin < planeDim; yin++ {
				for xin := 0; xin < planeDim; xin++ {
					xrot := (float64(xin)-u)*math.Cos(angle) + (u-float64(yin))*math.Sin(angle)
					yrot := -(float64(xin)-u)*math.Sin(angle) + (u-float64(yin))*math.Cos(angle)
					ytranslated := -yrot + u
					xtranslated := -xrot + u
					if (ytranslated >= 0) && (ytranslated < planeDim) &&
						(xtranslated >= 0) && (xtranslated < planeDim) {
						densityRotated[byte(ytranslated)][byte(xtranslated)] = geo.density[plane][yin][xin]
					}
				}
			}

			// Clear this axis plane in geo.density before copying the rotated densities
			for i := range geo.density[plane] {
				for j := range geo.density[plane][i] {
					geo.density[plane][i][j] = 0
				}
			}

			// Copy the rotated densities to the cleared axis plane in geo.density
			// Filter the rotated density with neighbor average using 3x3 kernel
			var sum byte = 0
			for i := 1; i < planeDim-1; i++ {
				for j := 1; j < planeDim-1; j++ {
					geo.density[plane][i][j] = densityRotated[i][j]
					if densityRotated[i][j] == 0 {
						sum = 0
						for m := i - 1; m <= i+1; m++ {
							for n := j - 1; n <= j+1; n++ {
								sum += densityRotated[m][n]
							}
						}
						geo.density[plane][i][j] = sum / 8
					}
				}
			}
		}
	// axis j
	case "j":
		// loop over the planes
		for plane := 0; plane < planeDim; plane++ {

			// Clear the temporary storage for every plane rotation
			for i := range densityRotated {
				for j := range densityRotated[i] {
					densityRotated[i][j] = 0
				}
			}

			// rotate angle and store in temp storage
			// remove the mean to rotate, put it back after
			for yin := 0; yin < planeDim; yin++ {
				for xin := 0; xin < planeDim; xin++ {
					xrot := (float64(xin)-u)*math.Cos(angle) + (u-float64(yin))*math.Sin(angle)
					yrot := -(float64(xin)-u)*math.Sin(angle) + (u-float64(yin))*math.Cos(angle)
					ytranslated := -yrot + u
					xtranslated := -xrot + u
					if (ytranslated >= 0) && (ytranslated < planeDim) &&
						(xtranslated >= 0) && (xtranslated < planeDim) {
						densityRotated[byte(ytranslated)][byte(xtranslated)] = geo.density[yin][plane][xin]
					}
				}
			}

			// Clear this axis plane in geo.density before copying the rotated densities
			for i := range geo.density[plane] {
				for j := range geo.density[plane][i] {
					geo.density[i][plane][j] = 0
				}
			}

			// Copy the rotated densities to the cleared axis plane in geo.density
			// Filter the rotated density with neighbor average using 3x3 kernel
			var sum byte = 0
			for i := 1; i < planeDim-1; i++ {
				for j := 1; j < planeDim-1; j++ {
					geo.density[i][plane][j] = densityRotated[i][j]
					if densityRotated[i][j] == 0 {
						sum = 0
						for m := i - 1; m <= i+1; m++ {
							for n := j - 1; n <= j+1; n++ {
								sum += densityRotated[m][n]
							}
						}
						geo.density[i][plane][j] = sum / 8
					}
				}
			}
		}
	// axis k
	case "k":
		// loop over the planes
		for plane := 0; plane < planeDim; plane++ {

			// Clear the temporary storage for every plane rotation
			for i := range densityRotated {
				for j := range densityRotated[i] {
					densityRotated[i][j] = 0
				}
			}

			// rotate angle and store in temp storage
			// remove the mean to rotate, put it back after
			for yin := 0; yin < planeDim; yin++ {
				for xin := 0; xin < planeDim; xin++ {
					xrot := (float64(xin)-u)*math.Cos(angle) + (u-float64(yin))*math.Sin(angle)
					yrot := -(float64(xin)-u)*math.Sin(angle) + (u-float64(yin))*math.Cos(angle)
					ytranslated := -yrot + u
					xtranslated := -xrot + u
					if (ytranslated >= 0) && (ytranslated < planeDim) &&
						(xtranslated >= 0) && (xtranslated < planeDim) {
						densityRotated[byte(ytranslated)][byte(xtranslated)] = geo.density[yin][xin][plane]
					}
				}
			}

			// Clear this axis plane in geo.density before copying the rotated densities
			for i := range geo.density[plane] {
				for j := range geo.density[plane][i] {
					geo.density[i][j][plane] = 0
				}
			}

			// Copy the rotated densities to the cleared axis plane in geo.density
			// Filter the rotated density with neighbor average using 3x3 kernel
			var sum byte = 0
			for i := 1; i < planeDim-1; i++ {
				for j := 1; j < planeDim-1; j++ {
					geo.density[i][j][plane] = densityRotated[i][j]
					if densityRotated[i][j] == 0 {
						sum = 0
						for m := i - 1; m <= i+1; m++ {
							for n := j - 1; n <= j+1; n++ {
								sum += densityRotated[m][n]
							}
						}
						geo.density[i][j][plane] = sum / 8
					}
				}
			}
		}
	}
	return nil
}

// Reload the geometric object
func (geo *Geometric) reloadGeometricObject(f *os.File) error {
	// Read the geometric object file containing the densities
	// reset the file descriptor to start
	f.Seek(0, io.SeekStart)
	for i := range planeDim {
		for j := range planeDim {
			for k := range planeDim - 1 {
				_, err := fmt.Fscanf(f, "%d", &geo.density[i][j][k])
				if err != nil {
					fmt.Printf("Fscanf for densities[%d][%d][%d] error: %v\n", i, j, k, err.Error())
					return fmt.Errorf("function Fscanf for densities[%d][%d][%d] error: %v", i, j, k, err.Error())
				}
			}
			_, err := fmt.Fscanf(f, "%d\n", &geo.density[i][j][planeDim-1])
			if err != nil {
				fmt.Printf("Fscanf for densities[%d][%d] newline error: %v\n", i, j, err.Error())
				return fmt.Errorf("function Fscanf for densities[%d][%d] newline error: %v", i, j, err.Error())
			}
		}
	}

	return nil
}

// get min sq error for this plane
func (geo *Geometric) getPlaneMassError(class int, axis int, plane int, planeErrorChan chan<- float64) {

	// get the bounds (number of rows and columns) for this plane
	rowShifts := geo.geoRefDims[class][axis][plane].nrows / 2
	colShifts := geo.geoRefDims[class][axis][plane].ncols / 2
	minSqErr := math.MaxFloat64
	// shift the reference over the sample and find the shift having the min sq error
	// the allowable number of shifts is determined by the reference bounds
	switch axis {
	case 0:
		for i := range rowShifts {
			for j := range colShifts {
				sqErr := 0
				// sum the rows and find the squared difference from reference
				for k := range geo.geoRefDims[class][axis][plane].nrows {
					rowsum := 0
					for m := range geo.geoRefDims[class][axis][plane].ncols {
						rowsum += int(geo.density[plane][k+i][m+j])
					}
					diff := geo.geoRefMass[axis][plane].row[k] - rowsum
					sqErr += diff * diff
				}
				// sum the columns and find the squared difference from reference
				for m := range geo.geoRefDims[class][axis][plane].ncols {
					colsum := 0
					for k := range geo.geoRefDims[class][axis][plane].nrows {
						colsum += int(geo.density[plane][k+i][m+j])
					}
					diff := geo.geoRefMass[axis][plane].col[m] - colsum
					sqErr += diff * diff
				}
				if float64(sqErr) < minSqErr {
					minSqErr = float64(sqErr)
				}
			}
		}
		// normalize square error by the size of the reference
		// send the normalized min square error to caller
		planeErrorChan <- minSqErr / float64(geo.geoRefDims[class][axis][plane].nrows*geo.geoRefDims[class][axis][plane].ncols)
	case 1:
		for i := range rowShifts {
			for j := range colShifts {
				sqErr := 0
				// sum the rows and find the squared difference from reference
				for k := range geo.geoRefDims[class][axis][plane].nrows {
					rowsum := 0
					for m := range geo.geoRefDims[class][axis][plane].ncols {
						rowsum += int(geo.density[k+i][plane][m+j])
					}
					diff := geo.geoRefMass[axis][plane].row[k] - rowsum
					sqErr += diff * diff
				}
				// sum the columns and find the squared difference from reference
				for m := range geo.geoRefDims[class][axis][plane].ncols {
					colsum := 0
					for k := range geo.geoRefDims[class][axis][plane].nrows {
						colsum += int(geo.density[k+i][plane][m+j])
					}
					diff := geo.geoRefMass[axis][plane].col[m] - colsum
					sqErr += diff * diff
				}
				if float64(sqErr) < minSqErr {
					minSqErr = float64(sqErr)
				}
			}
		}
		// normalize square error by the size of the reference
		// send the normalized min square error to caller
		planeErrorChan <- minSqErr / float64(geo.geoRefDims[class][axis][plane].nrows*geo.geoRefDims[class][axis][plane].ncols)
	case 2:
		for i := range rowShifts {
			for j := range colShifts {
				sqErr := 0
				// sum the rows and find the squared difference from reference
				for k := range geo.geoRefDims[class][axis][plane].nrows {
					rowsum := 0
					for m := range geo.geoRefDims[class][axis][plane].ncols {
						rowsum += int(geo.density[k+i][m+j][plane])
					}
					diff := geo.geoRefMass[axis][plane].row[k] - rowsum
					sqErr += diff * diff
				}
				// sum the columns and find the squared difference from reference
				for m := range geo.geoRefDims[class][axis][plane].ncols {
					colsum := 0
					for k := range geo.geoRefDims[class][axis][plane].nrows {
						colsum += int(geo.density[k+i][m+j][plane])
					}
					diff := geo.geoRefMass[axis][plane].col[m] - colsum
					sqErr += diff * diff
				}
				if float64(sqErr) < minSqErr {
					minSqErr = float64(sqErr)
				}
			}
		}
		// normalize square error by the size of the reference
		// send the normalized min square error to caller
		planeErrorChan <- minSqErr / float64(geo.geoRefDims[class][axis][plane].nrows*geo.geoRefDims[class][axis][plane].ncols)
	default:

	}
}

// compute min square error for all planes in this axis and return via channel
func (geo *Geometric) getAxisMassError(class, axis int, axisErrorChan chan<- float64) {
	// loop over planes and get plane mass errors using goroutines and channel
	planeErrorChan := make(chan float64)
	for plane := range planeDim {
		// each goroutine finds the minimum square error for its assigned plane
		go geo.getPlaneMassError(class, axis, plane, planeErrorChan)
	}

	// sum the plane square errors
	minSqError := 0.0
	for range planeDim {
		minSqError += <-planeErrorChan
	}

	// send the plane square error to caller
	axisErrorChan <- minSqError
}

// classify the geometric objects

func (geo *Geometric) classifyGeometric() error {
	// communicate results of mass error from each axis/plane
	axisError := make(chan float64)

	// loop over the number of samples
	for range geo.nsamples {
		// min sq mass error
		minSqError := math.MaxFloat64
		// class with min sq error
		minClass := 0
		// generate a geometric object with noise level and shift using geoRefDims
		ngeometricObj := rand.Intn(classes)
		geometricObj := geometricObjects[ngeometricObj]
		geometricObject.CreateObject(geometricObj, geo.noiseLevel, true)
		// loop over geometric references and open one at a time
		for class, obj := range geometricObjects {
			// read geometric reference mass sums into memory for this reference only
			fgeoref, err := os.Open(filepath.Join(dataDir, obj+".txt"))
			if err != nil {
				fmt.Printf("open %s error: %v\n", geometricObj, err.Error())
				return fmt.Errorf("open %s error: %v", geometricObj, err.Error())
			}
			for j := range geo.geoRefMass {
				for k := range geo.geoRefMass[j][:planeDim-1] {
					fmt.Fscanf(fgeoref, "%d ", &geo.geoRefMass[j][k])
				}
				fmt.Fscanf(fgeoref, "%d\n", &geo.geoRefMass[j][planeDim-1])
			}
			// close file
			fgeoref.Close()

			// find minimum mass error over rowsums and colsums for all axes and planes
			// launch goroutines for each axis and each plane :  3*50 goroutines
			// use channel communication between goroutines
			// use geoRefDims for shifting the object inside the planes
			for axes := range naxes {
				go geo.getAxisMassError(class, axes, axisError)
			}

			sqerr := 0.0
			for range naxes {
				sqerr += <-axisError
			}
			if sqerr < minSqError {
				minSqError = sqerr
				minClass = class
			}
		}

		// store geometric class count and correct classification
		geo.statistics.classCount[ngeometricObj]++
		if minClass == ngeometricObj {
			geo.statistics.correct[ngeometricObj]++
		}
	}
	return nil
}

// runs classification on the geometric object
func handleGeometricClassification(w http.ResponseWriter, r *http.Request) {

	var (
		plot PlotT
		geo  *Geometric
		err  error
	)

	// get number of samples and noise level
	txt := r.FormValue("samples")
	if len(txt) == 0 {
		plot.Status = "Enter number of samples and noise level"
		// Write to HTTP using template and grid
		if err := tmplGeometricClassification.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}

	nsamples, err := strconv.Atoi(txt)
	if err != nil {
		fmt.Printf("conversion of samples to int error: %v\n", err.Error())
		plot.Status = fmt.Sprintf("conversion of samples to int error: %v", err.Error())
		// Write to HTTP using template and grid
		if err := tmplGeometricClassification.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}

	// Determine if the files containing the reference masses and dimensions exist
	_, err = os.Stat(filepath.Join(dataDir, geometricrefdims))
	if err != nil {
		// create geometric references
		geometricObject.CreateObject("geometricreferences", 0, false)
	}

	// Construct Geometric instance for classification
	geo, err = newGeometricClassification(r, &plot, nsamples)
	if err != nil {
		fmt.Printf("newGeometricClassification() error: %v\n", err)
		plot.Status = fmt.Sprintf("newGeometricClassification() error: %v", err.Error())
		// Write to HTTP using template and grid
		if err := tmplGeometricClassification.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}
	defer geo.fmass.Close()

	// generate samples and classify using the noise level and shift
	err = geo.classifyGeometric()
	if err != nil {
		fmt.Printf("classifyGeometric() error: %v\n", err)
		plot.Status = fmt.Sprintf("classifyGeometric() error: %v", err.Error())
		// Write to HTTP using template and grid
		if err := tmplGeometricClassification.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}

	// insert test results classification into plot table for display
	err = geo.tabulateTestResults()
	if err != nil {
		fmt.Printf("tabulateTestResults error: %v\n", err.Error())
		plot.Status = fmt.Sprintf("tabulateTestResults error: %v", err.Error())
		// Write to HTTP using template and grid
		if err := tmplGeometricClassification.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return

	}
	// Put data in PlotT
	err = geo.gridFillInterp()
	if err != nil {
		fmt.Printf("gridFillInterp() error: %v\n", err)
		plot.Status = fmt.Sprintf("gridFillInterp() error: %v", err.Error())
		// Write to HTTP using template and grid
		if err := tmplGeometricClassification.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}

	// insert x-labels and y-labels in PlotT
	geo.insertLabels()

	// Execute data on HTML template
	if err = tmplGeometricClassification.Execute(w, geo.plot); err != nil {
		log.Fatalf("Write to HTTP output using template with error: %v\n", err)
	}
}

// runs display on the geometric object
func handleGeometricDisplay(w http.ResponseWriter, r *http.Request) {

	var (
		plot PlotT
		geo  *Geometric
	)

	// Determine if a new geometric object is wanted
	txt := r.FormValue("newgeometric")
	if len(txt) > 0 {
		geometricObj := r.FormValue("geometricobject")
		txt = r.FormValue("noiselevel")
		if len(txt) == 0 {
			fmt.Println("enter noise level")
			plot.Status = "enter noise level"
			// Write to HTTP using template and grid
			if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}
		noiseLevel, err := strconv.Atoi(txt)
		if err != nil {
			fmt.Printf("noiseLevel integer conversion error: %v\n", err.Error())
			plot.Status = fmt.Sprintf("noiseLevel integer conversion error: %v\n", err.Error())
			// Write to HTTP using template and grid
			if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}
		txt = r.FormValue("shiftgeometric")
		shift := false
		if txt == "shiftgeometric" {
			shift = true
		}
		err = geometricObject.CreateObject(geometricObj, noiseLevel, shift)
		if err != nil {
			fmt.Printf("createObject() error: %v\n", err)
			plot.Status = fmt.Sprintf("createObject() error: %v", err.Error())
			// Write to HTTP using template and grid
			if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}
	}

	// Open the geometric object file containing the densities
	f, err := os.Open(filepath.Join(dataDir, geometricobject))
	if err != nil {
		fmt.Printf("Open file %s error: %v\n", geometricobject, err)
		plot.Status = "Check New Geometric Object and select the geometric object"
		// Write to HTTP using template and grid
		if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}
	defer f.Close()

	// create Geometric instance to hold state
	geo, err = newGeometricDisplay(r, &plot, f)
	if err != nil {
		fmt.Printf("newComputedTomography() error: %v\n", err)
		plot.Status = fmt.Sprintf("newComputedTomography error: %v", err.Error())
		// Write to HTTP using template and grid
		if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
			log.Fatalf("Write to HTTP output using template with error: %v\n", err)
		}
		return
	}

	// Check for a reload of the geometric object
	txt = r.FormValue("reset")
	if txt == "resetgeometricobj" {
		err = geo.reloadGeometricObject(f)
		if err != nil {
			fmt.Printf("resetGeometricObject error: %v\n", err.Error())
			plot.Status = "Rotation angle conversion error"
			// Write to HTTP using template and grid
			if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}
	}

	// Determine if rotate requested and perform the rotation of the axis planes
	rotateRad := 0.0
	rotate := r.FormValue("rotate")
	if rotate == "rotateplanes" {
		txt = r.FormValue("rotationangle")
		geo.rotateAngle, err = strconv.ParseFloat(txt, 64)
		if err != nil {
			fmt.Printf("Rotation angle conversion error: %v", err)
			plot.Status = fmt.Sprintf("Rotation angle conversion error: %v", err.Error())
			// Write to HTTP using template and grid
			if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		} else {
			rotateRad = deg2rad * geo.rotateAngle
		}
		geo.rotateAxis = r.FormValue("rotationaxis")
		if len(txt) == 0 {
			plot.Status = "Rotate axis not selected"
			// Write to HTTP using template and grid
			if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		} else {
			err = geo.rotatePlanes(geo.rotateAxis, rotateRad)
			if err != nil {
				plot.Status = "rotatePlanes error"
				// Write to HTTP using template and grid
				if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
					log.Fatalf("Write to HTTP output using template with error: %v\n", err)
				}
				return
			}
		}
	}

	// expand a plane for the geometric object
	txt = r.FormValue("zoom")
	if txt == "zoomaxisplane" {
		zoomAxis := r.FormValue("zoomaxis")
		txt = r.FormValue("zoomplane")
		zoomPlane, err := strconv.Atoi(txt)
		if err != nil {
			fmt.Printf("zoomPlane int conversion error: %v\n", err.Error())
			plot.Status = fmt.Sprintf("zoomPlane int conversion error: %v\n", err.Error())
			// Write to HTTP using template and grid
			if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}
		err = geo.processZoom(zoomAxis, zoomPlane)
		if err != nil {
			fmt.Printf("processZoom error: %v\n", err.Error())
			plot.Status = fmt.Sprintf("processZoom error: %v\n", err.Error())
			// Write to HTTP using template and grid
			if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}
		// show Overview of geometric object density
	} else {
		err := geo.processOverview()
		if err != nil {
			fmt.Printf("processOverview error: %v\n", err.Error())
			plot.Status = fmt.Sprintf("processOverview error: %v\n", err.Error())
			// Write to HTTP using template and grid
			if err := tmplGeometricDisplay.Execute(w, plot); err != nil {
				log.Fatalf("Write to HTTP output using template with error: %v\n", err)
			}
			return
		}
	}

	// Execute data on HTML template
	// Write to HTTP using template and grid
	if err := tmplGeometricDisplay.Execute(w, geo.plot); err != nil {
		log.Fatalf("Write to HTTP output using template with error: %v\n", err)
	}
}

// executive creates the HTTP handlers, listens and serves
func main() {
	// Set up HTTP servers with handlers for geometric classification and display

	// Create HTTP handler for performing CT
	http.HandleFunc(patterngeometricdisplay, handleGeometricDisplay)
	fmt.Printf("Computed Tomography Server listening on %v.\n", addr)
	// Create HTTP handler for testing
	http.HandleFunc(patterngeometricclassification, handleGeometricClassification)
	http.ListenAndServe(addr, nil)
}
