/*
Create geometric object densities for paraboloid, ellipsoid,
hyperbolic paraboloid, plane, cone, box, cylinder, and cube.
These can be surfaces or solids, depending on whether the
volume is convex.
*/

package geometricObject

import (
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
)

const (
	planeDim         = 50                    // plane dimension in cells in i, j, k data space
	geometricobject  = "geometricobject.txt" // 3D geometric object file containing the densities, 50x50x50
	dataDir          = "data/"               // directory for player positions
	geometricrefdims = "geometricrefdim.txt" // row/column dimensions of references
)

// 3D geometric object
type GeoObject struct {
	density    [][][]byte
	noiseLevel int
	shift      bool
}

// add noise to the density and shift the location of the geometric object
func (geo *GeoObject) addNoiseShift() error {
	// Save geo.density to a temp file
	const tempfile = "tempdensity.txt"
	ftemp, err := os.Create(filepath.Join(dataDir, tempfile))
	if err != nil {
		fmt.Printf("open file %s error: %v\n", tempfile, err.Error())
		return fmt.Errorf("open file %s error: %v", tempfile, err.Error())
	}
	defer ftemp.Close()

	for i := range planeDim {
		for j := range planeDim {
			for k := range planeDim - 1 {
				fmt.Fprintf(ftemp, "%d ", geo.density[i][j][k])
			}
			fmt.Fprintf(ftemp, "%d\n", geo.density[i][j][planeDim-1])
		}
	}

	// Rewind temp density file
	ftemp.Seek(0, io.SeekStart)

	// Clear geo.density
	for i := range planeDim {
		for j := range planeDim {
			for k := range planeDim {
				geo.density[i][j][k] = 0
			}
		}
	}

	// Read in density file and place in a shifted position with density noise
	// Read the geometric object file containing the densities
	// compute noise and shift

	signi := 1
	if sign := rand.IntN(2); sign > 0 {
		signi = -1
	}
	signj := 1
	if sign := rand.IntN(2); sign > 0 {
		signj = -1
	}
	signk := 1
	if sign := rand.IntN(2); sign > 0 {
		signk = -1
	}

	ishift := signi * rand.IntN(planeDim/4)
	jshift := signj * rand.IntN(planeDim/4)
	kshift := signk * rand.IntN(planeDim/4)

	deltaI := 0
	deltaJ := 0
	deltaK := 0
	junk := 0
	for i := range planeDim {
		deltaI = i + ishift
		for j := range planeDim {
			deltaJ = j + jshift
			for k := range planeDim - 1 {
				deltaK = k + kshift
				if (deltaI >= 0 && deltaI < planeDim) && (deltaJ >= 0 && deltaJ < planeDim) &&
					(deltaK >= 0 && deltaK < planeDim) {
					_, err := fmt.Fscanf(ftemp, "%d", &geo.density[deltaI][deltaJ][deltaK])
					if err != nil {
						fmt.Printf("Fscanf 1 for density[%d][%d][%d] error: %v\n", deltaI, deltaJ, deltaK, err.Error())
						return fmt.Errorf("function Fscanf 1 for density[%d][%d][%d] error: %v", deltaI, deltaJ, deltaK, err.Error())
					}
					// add noise, (+/-)noiseLevel/2
					noise := float64(geo.noiseLevel) * (rand.Float64() - 0.5)
					try := float64(geo.density[deltaI][deltaJ][deltaK]) + noise
					geo.density[deltaI][deltaJ][deltaK] = byte(min(max(0, try), 9.0))
				} else {
					fmt.Fscanf(ftemp, "%d", &junk)
				}
			}
			deltaK = planeDim - 1 + kshift
			if (deltaI >= 0 && deltaI < planeDim) && (deltaJ >= 0 && deltaJ < planeDim) &&
				(deltaK >= 0 && deltaK < planeDim) {
				_, err := fmt.Fscanf(ftemp, "%d\n", &geo.density[deltaI][deltaJ][deltaK])
				if err != nil {
					fmt.Printf("Fscanf 2 for density[%d][%d][%d] error: %v\n", deltaI, deltaJ, deltaK, err.Error())
					return fmt.Errorf("function Fscanf 2 for density[%d][%d][%d] error: %v", deltaI, deltaJ, deltaK, err.Error())
				}
				// add noise, (+/-)noiseLevel/2
				noise := float64(geo.noiseLevel) * (rand.Float64() - 0.5)
				try := float64(geo.density[deltaI][deltaJ][deltaK]) + noise
				geo.density[deltaI][deltaJ][deltaK] = byte(min(max(0, try), 9.0))
			} else {
				fmt.Fscanf(ftemp, "%d\n", &junk)
			}
		}
	}
	return nil
}

// plane, surface
func (geo *GeoObject) createPlane() {
	// choose center of plane (x1, y1, z1) = (25, 25, 25)
	// vary x, y, in (0, 49)
	// Normal to plane is Ai + Bj + Ck
	// dot product: A(x-x1) + B(y-y1) + C(z-z1) = 0
	// Ax + By + Cz = D, => D=A*x1+B*y1+C*z1
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	A := 2
	B := 3
	C := 4
	a := x1 / 2
	b := y1 / 2
	c := z1 / 2

	D := A*x1 + B*y1 + C*z1
	var black byte = 9

	for x := x1 - a; x < x1+a; x++ {
		for y := y1 - b; y < y1+b; y++ {
			z := (D - A*x - B*y) / C
			// this point is inside the data space and in the plane
			if z >= z1-c && z < z1+c {
				geo.density[x][y][z] = black
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}

}

// cardioid of revolution, surface
func (geo *GeoObject) createCardioidRevolution() {
	// use polar coordinates, (r, theta), 0<=theta<2pi
	// r=a(1-cos(theta)), rotate cardioid(r,theta) about x axis to create 3D surface
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := x1 / 2
	// one degree resolution
	del := math.Pi / 180.0
	shiftx := x1 + a - a/8
	var black byte = 9
	// loop over theta, 0<=theta<180
	theta := 0.0
	for range 180 {
		//   calculate r
		r := float64(a) * (1.0 - math.Cos(theta))
		// calculate x=r*cos(theta), for (+/-) theta
		xminus := r * math.Cos(-theta)
		xplus := r * math.Cos(theta)
		// calculate h=r*sin(theta)
		h := r * math.Sin(theta)
		phi := 0.0
		// loop over phi, 0<=phi<180
		for range 180 {
			// z=h*sin(phi), for (+/-) phi
			z := h * math.Sin(phi)
			y := h * math.Cos(phi)
			// calculate density for (+/-) theta and phi
			// translate x to (0,planeDim) with planeDim/2+a-a/8 = shiftx
			// translate (y,z) to (0,planeDim) with planeDim/2
			geo.density[y1+int(y)][z1+int(z)][int(xminus)+shiftx] = black
			geo.density[y1+int(y)][z1+int(z)][int(xplus)+shiftx] = black
			geo.density[y1+int(y)][z1+int(-z)][int(xminus)+shiftx] = black
			geo.density[y1+int(y)][z1+int(-z)][int(xplus)+shiftx] = black
			phi += del
		}
		theta += del
	}
	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// cardiod of revolution, solid with varying density
func (geo *GeoObject) createCardioidRevolutionSolid() {
	// use polar coordinates, (r, theta), 0<=theta<2pi
	// r=a(1-cos(theta)), rotate cardioid(r,theta) about x axis to create 3D surface
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := x1 / 2
	norm := float64(2 * a)
	var density byte
	// one degree resolution
	del := math.Pi / 180.0
	shiftx := x1 + a - a/8 - 1
	black := 9.0
	nrsteps := 3
	rstep := 1.0 / float64(nrsteps)
	for i := a; i > 0; i-- {
		// loop over theta, 0<=theta<180
		theta := 0.0
		for range 180 {
			for n := range nrsteps {
				// calculate r
				k := float64(i) - float64(n)*rstep
				r := float64(k) * (1.0 - math.Cos(theta))
				// calculate x=r*cos(theta), for (+/-) theta
				xminus := r * math.Cos(-theta)
				xplus := r * math.Cos(theta)
				// calculate h=r*sin(theta)
				h := r * math.Sin(theta)
				phi := 0.0
				// loop over phi, 0<=phi<180
				for range 180 {
					// z=h*sin(phi), for (+/-) phi
					z := h * math.Sin(phi)
					y := h * math.Cos(phi)
					// calculate density for (+/-) theta and phi
					// translate x to (0,planeDim) with planeDim/2+a-a/8 = shiftx
					// translate (y,z) to (0,planeDim) with planeDim/2
					// center is the most dense, decreasing as you move away from center
					density = byte(black * (1.0 - r/norm))
					geo.density[y1+int(y)][z1+int(z)][int(xminus)+shiftx] = density
					geo.density[y1+int(y)][z1+int(z)][int(xplus)+shiftx] = density
					geo.density[y1+int(y)][z1+int(-z)][int(xminus)+shiftx] = density
					geo.density[y1+int(y)][z1+int(-z)][int(xplus)+shiftx] = density
					phi += del
				}
			}
			theta += del
		}
	}
	geo.density[y1][z1][shiftx] = byte(black)

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}

}

// lemniscate of revolution, surface
func (geo *GeoObject) createLemniscateRevolution() {
	// use polar coordinates, (r, theta), 0<=theta<2pi
	// r^2=2*a^2*cos(2*theta), rotate lemniscate(r,theta) about x-axis to create 3D surface
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := x1 / 2
	// use 1/2 degree resolution
	del := math.Pi / 360.0
	var black byte = 9
	// loop over theta, 0<=theta<45, and use symmetry to find other values
	theta := 0.0
	for range 90 {
		// calculate r
		r := float64(a) * math.Sqrt(math.Cos(2.0*theta))
		// calculate x=r*cos(theta), first quadrant, use symmetry for others
		x := r * math.Cos(theta)
		// calculate h=r*sin(theta)
		h := r * math.Sin(theta)
		phi := 0.0
		// loop over phi, 0<=phi<90
		for range 180 {
			// z=h*sin(phi), for (+/-) phi
			// y=h*cos(phi), for (+/-) phi
			z := h * math.Sin(phi)
			y := h * math.Cos(phi)
			// calculate density for (+/-) theta and phi
			// translate (x,y,z) to (0,planeDim) with planeDim/2
			geo.density[y1+int(y)][z1+int(z)][int(x)+x1] = black
			geo.density[y1+int(y)][z1+int(z)][int(-x)+x1] = black

			geo.density[y1+int(-y)][z1+int(z)][int(x)+x1] = black
			geo.density[y1+int(-y)][z1+int(z)][int(-x)+x1] = black

			geo.density[y1+int(y)][z1+int(-z)][int(x)+x1] = black
			geo.density[y1+int(y)][z1+int(-z)][int(-x)+x1] = black

			geo.density[y1+int(-y)][z1+int(-z)][int(x)+x1] = black
			geo.density[y1+int(-y)][z1+int(-z)][int(-x)+x1] = black

			phi += del
		}
		theta += del
	}
	geo.density[y1][z1][x1] = black

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}

}

// lemniscate of revolution, solid with varying density
func (geo *GeoObject) createLemniscateRevolutionSolid() {
	// use polar coordinates, (r, theta), 0<=theta<2pi
	// r^2=2*a^2*cos(2*theta), rotate lemniscate(r,theta) about x-axis to create 3D surface
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := 3 * x1 / 4
	// use one degree resolution
	del := math.Pi / 180.0
	black := 9.0
	nrsteps := 3
	rstep := 1.0 / float64(nrsteps)
	norm := float64(a)
	var density byte
	for i := a; i > 0; i-- {
		// loop over theta, 0<=theta<45, and use symmetry to find other values
		theta := 0.0
		for range 45 {
			for n := range nrsteps {
				// calculate r
				k := float64(i) - float64(n)*rstep
				r := k * math.Sqrt(math.Cos(2.0*theta))
				// calculate x=r*cos(theta), first quadrant, use symmetry for others
				x := r * math.Cos(theta)
				// calculate h=r*sin(theta)
				h := r * math.Sin(theta)
				phi := 0.0
				// loop over phi, 0<=phi<90
				for range 90 {
					// z=h*sin(phi), for (+/-) phi
					// y=h*cos(phi), for (+/-) phi
					z := h * math.Sin(phi)
					y := h * math.Cos(phi)
					density = byte(black * (1.0 - r/norm))
					// calculate density for (+/-) theta and phi
					// translate (x,y,z) to (0,planeDim) with planeDim/2
					geo.density[y1+int(y)][z1+int(z)][int(x)+x1] = density
					geo.density[y1+int(y)][z1+int(z)][int(-x)+x1] = density

					geo.density[y1+int(-y)][z1+int(z)][int(x)+x1] = density
					geo.density[y1+int(-y)][z1+int(z)][int(-x)+x1] = density

					geo.density[y1+int(y)][z1+int(-z)][int(x)+x1] = density
					geo.density[y1+int(y)][z1+int(-z)][int(-x)+x1] = density

					geo.density[y1+int(-y)][z1+int(-z)][int(x)+x1] = density
					geo.density[y1+int(-y)][z1+int(-z)][int(-x)+x1] = density
					phi += del
				}
			}
			theta += del
		}
	}
	geo.density[y1][z1][x1] = byte(black)

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// Four-leaved rose of revolution, surface
func (geo *GeoObject) createRose4LeafRevolution() {
	// r=4*sin(2*theta), rotate Rose4Leaf(r,theta) about x-axis to create 3D surface
	// use polar coordinates, (r, theta), 0<=theta<2pi
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	// one degree resolution
	del := math.Pi / 180.0
	var black byte = 9
	// loop over theta, 0<=theta<90, and use symmetry to find other values
	theta := 0.0
	for range 90 {
		// calculate r
		r := float64(3*x1/4) * math.Sin(2.0*theta)
		// calculate x=r*cos(theta), first quadrant, use symmetry for others
		x := r * math.Cos(theta)
		// calculate h=r*sin(theta)
		h := r * math.Sin(theta)
		phi := 0.0
		// loop over phi, 0<=phi<180
		for range 180 {
			// z=h*sin(phi), for (+/-) phi
			// y=h*cos(phi), for (+/-) phi
			z := h * math.Sin(phi)
			y := h * math.Cos(phi)
			// calculate density for (+/-) theta and phi
			// translate (x,y,z) to (0,planeDim) with planeDim/2
			geo.density[y1+int(y)][z1+int(z)][int(x)+x1] = black
			geo.density[y1+int(y)][z1+int(-z)][int(x)+x1] = black
			geo.density[y1+int(y)][z1+int(z)][int(-x)+x1] = black
			geo.density[y1+int(y)][z1+int(-z)][int(-x)+x1] = black
			phi += del
		}
		theta += del
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// Four-leaved rose of revolution, solid with varying density
func (geo *GeoObject) createRose4LeafRevolutionSolid() {
	// r=4*sin(2*theta), rotate Rose4Leaf(r,theta) about x-axis to create 3D surface
	// use polar coordinates, (r, theta), 0<=theta<2pi
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := 3 * x1 / 4
	// one degree resolution
	del := math.Pi / 180.0
	black := 9.0

	nrsteps := 3
	rstep := 1.0 / float64(nrsteps)
	norm := float64(a)
	var density byte
	for i := a; i >= 0; i-- {
		// loop over theta, 0<=theta<90, and use symmetry to find other values
		theta := 0.0
		for range 90 {
			for n := range nrsteps {
				k := float64(i) - float64(n)*rstep
				// calculate r
				r := float64(k) * math.Sin(2.0*theta)
				// calculate x=r*cos(theta), first quadrant, use symmetry for others
				x := r * math.Cos(theta)
				// calculate h=r*sin(theta)
				h := r * math.Sin(theta)
				phi := 0.0
				// loop over phi, 0<=phi<180
				for range 180 {
					// z=h*sin(phi), for (+/-) phi
					// y=h*cos(phi), for (+/-) phi
					z := h * math.Sin(phi)
					y := h * math.Cos(phi)
					// calculate density for (+/-) theta and phi
					// translate (x,y,z) to (0,planeDim) with planeDim/2
					density = byte(black * (1.0 - r/norm))
					geo.density[y1+int(y)][z1+int(z)][int(x)+x1] = density
					geo.density[y1+int(y)][z1+int(-z)][int(x)+x1] = density
					geo.density[y1+int(y)][z1+int(z)][int(-x)+x1] = density
					geo.density[y1+int(y)][z1+int(-z)][int(-x)+x1] = density
					phi += del
				}
			}
			theta += del
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// potential well, surface, amount of work required to move from 1 to r
// in height above earth
func (geo *GeoObject) createPotentialWell() {
	// use polar coordiates, (r,theta), r>=1, 0<=theta<2pi
	// w = k*(1-1/r), k=planeDim/2
	var black byte = 9
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	rmax := 3 * z1 / 4
	z1c := z1 + rmax/2 - 1
	del := 2.0 * math.Pi / 360.0
	// loop r from 1 to planeDim/2
	for r := 1; r <= rmax; r++ {
		//   loop theta from 0 to 2pi
		theta := 0.0
		for range 360 {
			//   x=r*cos(theta), y=r*sin(theta), z=k*(1-1/r)
			x := float64(r) * math.Cos(theta)
			y := float64(r) * math.Sin(theta)
			z := float64(r) * (1.0 - 1.0/float64(r))
			//   center x,y,z in (0,50) by adding planeDim/2
			geo.density[z1c-int(z)][x1+int(x)][y1+int(y)] = black
			theta += del
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// elliptic cylinder, surface
func (geo *GeoObject) createCylinder() {
	// center (x1,y1,z1)
	// (x)^2/a^2 + (y)^2/b^2 = 1
	var black byte = 9
	eps := .1
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := x1/2 - 3
	a2 := a * a
	b := y1/2 + 3
	b2 := b * b
	c := z1 / 2
	for x := -a; x <= a; x++ {
		for y := -b; y <= b; y++ {
			diff := 1.0 - float64(x*x)/float64(a2) - float64(y*y)/float64(b2)
			if diff > -eps && diff < eps {
				for z := -c; z <= c; z++ {
					geo.density[x+x1][y+y1][z+z1] = black
				}
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}

}

// elliptic cylinder, solid
func (geo *GeoObject) createCylinderSolid() {
	// center (x1,y1,z1)
	// (x)^2/a^2 + (y)^2/b^2 = 1
	black := 9.0
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := x1/2 - 3
	b := y1/2 + 3
	c := z1 / 2
	var density byte
	norm := float64(a*a + b*b + c*c)
	// Fill in the ellipsoid while any axis is greater than or equal to zero
	for i := a; i >= 0; i-- {
		a2 := i * i
		for j := b; j >= 0; j-- {
			b2 := j * j
			for x := -i; x <= i; x++ {
				for y := -j; y <= j; y++ {
					diff := 1.0 - float64(x*x)/float64(a2) - float64(y*y)/float64(b2)
					if diff >= 0 {
						for z := -c; z <= c; z++ {
							sumsq := float64(x*x + y*y + z*z)
							// center is the most dense, decreasing as you move away from center
							density = byte(black * (1.0 - math.Sqrt(sumsq/norm)))
							geo.density[x+x1][y+y1][z+z1] = density
						}
					}
				}
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// hyperbolic paraboloid, surface
func (geo *GeoObject) createHyperbolicParaboloid() {
	// y^2/b^2 - x^2/a^2 = z/c
	var black byte = 9
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := x1 / 2
	a2 := a * a
	b := y1 / 2
	b2 := b * b
	c := z1 / 2
	for x := -a; x <= a; x++ {
		for y := -b; y <= b; y++ {
			z := int((float64(y*y)/float64(b2) - float64(x*x)/float64(a2)) * float64(c))
			if z >= -z1 && z <= z1 {
				geo.density[z1+z][x1+x][y1+y] = black
			}
		}
	}

	for y := -b; y <= b; y++ {
		for z := -c; z <= c; z++ {
			tmp := float64(y*y)/float64(b2) - float64(z)/float64(c)
			if tmp >= 0 {
				x := int((math.Sqrt(tmp * float64(a2))))
				geo.density[z1+z][x1+x][y1+y] = black
			}
		}
	}

	for z := -c; z <= c; z++ {
		for x := -a; x <= a; x++ {
			tmp := float64(z)/float64(c) + float64(x*x)/float64(a2)
			if tmp >= 0 {
				y := int(math.Sqrt(tmp * float64(b2)))
				geo.density[z1+z][x1+x][y1+y] = black
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// cube, solid with decreasing density from center
func (geo *GeoObject) createCube() {
	black := 9.0
	// choose center of cube (x1,y1,z1) = (25,25,25)
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	del := planeDim / 4
	norm := 3.0 * float64(del*del)
	for x := x1 - del; x < x1+del; x++ {
		for y := y1 - del; y < y1+del; y++ {
			for z := z1 - del; z < z1+del; z++ {
				delx := float64(x - x1)
				dely := float64(y - y1)
				delz := float64(z - z1)
				sumsq := delx*delx + dely*dely + delz*delz
				// center is the most dense, decreasing as you move away from center
				geo.density[x][y][z] = byte(black * (1.0 - math.Sqrt(sumsq/norm)))
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}

}

// box, surface
func (geo *GeoObject) createBox() {
	var black byte = 9
	// choose center of box (x1,y1,z1) = (25,25,25)
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	del := planeDim / 4
	for x := x1 - del; x <= x1+del; x++ {
		for y := y1 - del; y <= y1+del; y++ {
			geo.density[x][y][z1-del] = black
			geo.density[x][y][z1+del] = black
		}
	}
	for y := y1 - del; y <= y1+del; y++ {
		for z := z1 - del; z <= z1+del; z++ {
			geo.density[x1-del][y][z] = black
			geo.density[x1+del][y][z] = black
		}
	}
	for x := x1 - del; x <= x1+del; x++ {
		for z := z1 - del; z <= z1+del; z++ {
			geo.density[x][y1-del][z] = black
			geo.density[x][y1+del][z] = black
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// ellipsoid, surface
func (geo *GeoObject) createEllipsoid() {
	// center (x1,y1,z1)
	// (x)^2/a^2 + (y)^2/b^2 + (z)^2/c^2 = 1
	var black byte = 9
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := x1/2 - 3
	a2 := a * a
	b := y1 / 2
	b2 := b * b
	c := z1/2 + 2
	c2 := c * c
	for x := -a; x <= a; x++ {
		for y := -b; y <= b; y++ {
			tmp := 1.0 - float64(x*x)/float64(a2) - float64(y*y)/float64(b2)
			if tmp >= 0 {
				z := int(math.Sqrt(tmp * float64(c2)))
				geo.density[x+x1][y+y1][z+z1] = black
				geo.density[x+x1][y+y1][-z+z1] = black
			}
		}
	}

	for y := -b; y <= b; y++ {
		for z := -c; z <= c; z++ {
			tmp := 1.0 - float64(z*z)/float64(c2) - float64(y*y)/float64(b2)
			if tmp >= 0 {
				x := int(math.Sqrt(tmp * float64(a2)))
				geo.density[x+x1][y+y1][z+z1] = black
				geo.density[-x+x1][y+y1][z+z1] = black
			}
		}
	}

	for z := -c; z <= c; z++ {
		for x := -a; x <= a; x++ {
			tmp := 1.0 - float64(z*z)/float64(c2) - float64(x*x)/float64(a2)
			if tmp >= 0 {
				y := int(math.Sqrt(tmp * float64(b2)))
				geo.density[x+x1][y+y1][z+z1] = black
				geo.density[x+x1][-y+y1][z+z1] = black
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// ellipsoid, solid with varying density
func (geo *GeoObject) createEllipsoidSolid() {
	// center (x1,y1,z1)
	// (x)^2/a^2 + (y)^2/b^2 + (z)^2/c^2 = 1
	black := 9.0
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	// assign axis maximums
	a := x1/2 - 4
	b := y1 / 2
	c := z1/2 + 4
	norm := float64(a*a + b*b + c*c)
	var density byte
	// Fill in the ellipsoid while any axis is greater than or equal to zero
	for i := a; i >= 0; i-- {
		a2 := i * i
		for j := b; j >= 0; j-- {
			b2 := j * j
			for k := c; k >= 0; k-- {
				c2 := k * k
				for x := -i; x <= i; x++ {
					for y := -j; y <= j; y++ {
						tmp := 1.0 - float64(x*x)/float64(a2) - float64(y*y)/float64(b2)
						if tmp >= 0 {
							z := int(math.Sqrt(tmp * float64(c2)))
							sumsq := float64(x*x + y*y + z*z)
							// center is the most dense, decreasing as you move away from center
							density = byte(black * (1.0 - math.Sqrt(sumsq/norm)))
							geo.density[x+x1][y+y1][z1+z] = density
							geo.density[x+x1][y+y1][z1-z] = density
						}
					}
				}
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// elliptic cone, surface
func (geo *GeoObject) createCone() {
	// center (x1,y1,z1)
	// (x)^2/a^2 + (y)^2/b^2 = (z)^2/c^2
	var black byte = 9
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := x1 / 2
	a2 := a * a
	b := y1 / 2
	b2 := b * b
	c := z1
	c2 := c * c
	z1c := z1 + c/2
	for x := -a; x <= a; x++ {
		for y := -b; y <= b; y++ {
			z := int(math.Sqrt((float64(x*x)/float64(a2) + float64(y*y)/float64(b2)) * float64(c2)))
			geo.density[z1c-z][x1+x][y1+y] = black
		}
	}

	for y := -b; y <= b; y++ {
		for z := 0; z <= c; z++ {
			tmp := float64(z*z)/float64(c2) - float64(y*y)/float64(b2)
			if tmp >= 0 {
				x := int(math.Sqrt(tmp * float64(a2)))
				geo.density[z1c-z][x1+x][y1+y] = black
				geo.density[z1c-z][x1-x][y1+y] = black
			}
		}
	}

	for z := 0; z <= c; z++ {
		for x := -a; x <= a; x++ {
			tmp := float64(z*z)/float64(c2) - float64(x*x)/float64(a2)
			if tmp >= 0 {
				y := int(math.Sqrt(tmp * float64(b2)))
				geo.density[z1c-z][x1+x][y1+y] = black
				geo.density[z1c-z][x1+x][y1-y] = black
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// elliptic cone, solid
func (geo *GeoObject) createConeSolid() {
	// center (x1,y1,z1)
	// (x)^2/a^2 + (y)^2/b^2 = (z)^2/c^2
	var black = 9.0
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := x1 / 2
	a2 := a * a
	b := y1 / 2
	b2 := b * b
	c := z1
	c2 := c * c
	z1c := z1 + c/2
	norm := float64(a*a + b*b + (c/2)*(c/2))
	var density byte
	/**************************************************************************/
	// Fill in the cone while any axis is greater than or equal to zero
	for i := a; i > 0; i-- {
		a2 := i * i
		for j := b; j > 0; j-- {
			b2 := j * j
			for k := c; k > 0; k-- {
				k2 := k * k
				k1c := z1 + k/2
				for x := -i; x <= i; x++ {
					for y := -j; y <= j; y++ {
						z := int(math.Sqrt((float64(x*x)/float64(a2) + float64(y*y)/float64(b2)) * float64(k2)))
						if z >= 0 && z <= k {
							sumsq := float64(x*x + y*y + (z-k/2)*(z-k/2))
							// center is the most dense, decreasing as you move away from center
							density = byte(black * (1.0 - math.Sqrt(sumsq/norm)))
							geo.density[k1c-z][x+x1][y+y1] = density
						}
					}
				}
			}
		}
	}

	//Miscellaneous problems
	for z := -c / 2; z <= c/2; z++ {
		x := 0
		y := 0
		sumsq := float64(x*x + y*y + z*z)
		density = byte(black * (1.0 - math.Sqrt(sumsq/norm)))
		geo.density[z1+z][x+x1][y+y1] = density
	}

	for z := 0; z < c; z++ {
		for x := -x1; x < x1; x++ {
			for y := -y1; y < y1; y++ {
				test := math.Sqrt((float64(x*x)/float64(a2) + float64(y*y)/float64(b2)) * float64(c2))
				if test > float64(z) {
					geo.density[z1c-z][x1+x][y1+y] = 0
				}
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// elliptic parabaloid, surface
func (geo *GeoObject) createParaboloid() {
	// center (x1,y1,z1)
	// x^2/a^2 + y^2/b^2 = z/c
	var black byte = 9
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	a := x1 / 2
	a2 := a * a
	b := y1 / 2
	b2 := b * b
	c := z1
	z1c := z1 + c/2
	for x := -a; x < a; x++ {
		for y := -b; y < b; y++ {
			z := int((float64(x*x)/float64(a2) + float64(y*y)/float64(b2)) * float64(c))
			if z1c >= z {
				geo.density[z1c-z][x+x1][y+y1] = black
			}
		}
	}

	for y := -b; y < b; y++ {
		for z := 0; z <= c; z++ {
			tmp := float64(z)/float64(c) - float64(y*y)/float64(b2)
			if tmp >= 0 {
				x := int((math.Sqrt(tmp * float64(a2))))
				geo.density[z1c-z][x1+x][y1+y] = black
				geo.density[z1c-z][x1-x][y1+y] = black
			}
		}
	}

	for z := 0; z <= c; z++ {
		for x := -a; x <= a; x++ {
			tmp := float64(z)/float64(c) - float64(x*x)/float64(a2)
			if tmp >= 0 {
				y := int(math.Sqrt(tmp * float64(b2)))
				geo.density[z1c-z][x1+x][y1+y] = black
				geo.density[z1c-z][x1+x][y1-y] = black
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// create a solid paraboloid with varying density
func (geo *GeoObject) createParaboloidSolid() {
	// center (x1,y1,z1)
	// x^2/a^2 + y^2/b^2 = z/c
	black := 9.0
	x1 := planeDim / 2
	y1 := planeDim / 2
	z1 := planeDim / 2
	// assign axis maximums
	a := x1 / 2
	b := y1 / 2
	c := z1
	//c12 := c / 2
	z1c := z1 + c/2
	norm := float64(a*a + b*b + (c/2)*(c/2))
	var density byte

	/**************************************************************************/
	// Fill in the paraboloid while any axis is greater than or equal to zero
	for i := a; i > 0; i-- {
		a2 := i * i
		for j := b; j > 0; j-- {
			b2 := j * j
			for k := c; k > 0; k-- {
				k1c := z1 + k/2
				for x := -i; x <= i; x++ {
					for y := -j; y <= j; y++ {
						z := int((float64(x*x)/float64(a2) + float64(y*y)/float64(b2)) * float64(k))
						if z >= 0 && z <= k {
							sumsq := float64(x*x + y*y + (z-k/2)*(z-k/2))
							// center is the most dense, decreasing as you move away from center
							density = byte(black * (1.0 - math.Sqrt(sumsq/norm)))
							geo.density[k1c-z][x+x1][y+y1] = density
						}
					}
				}
			}
		}
	}

	//Miscellaneous problems
	for z := -c / 2; z <= c/2; z++ {
		x := 0
		y := 0
		sumsq := float64(x*x + y*y + z*z)
		density = byte(black * (1.0 - math.Sqrt(sumsq/norm)))
		geo.density[z1+z][x+x1][y+y1] = density
	}

	a2 := a * a
	b2 := b * b
	for z := 0; z < c; z++ {
		for x := -x1; x < x1; x++ {
			for y := -y1; y < y1; y++ {
				test := (float64(x*x)/float64(a2) + float64(y*y)/float64(b2)) * float64(c)
				if test > float64(z) {
					geo.density[z1c-z][x1+x][y1+y] = 0
				}
			}
		}
	}

	// add noise to this geometric object and shift its location
	if geo.shift {
		// find maximum shift and choose random value in that range
		err := geo.addNoiseShift()
		if err != nil {
			fmt.Printf("addNoiseShift error: %v\n", err.Error())
		}
	}
}

// create geometric references consisting of plane row/column mass sums and plane dimensions
func (geo *GeoObject) createGeometricReferences() error {

	type Geo func()

	geometricObjects := []string{
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

	// functions that construct geometric objects
	geofunc := []Geo{geo.createEllipsoid, geo.createEllipsoidSolid, geo.createPlane,
		geo.createParaboloid, geo.createParaboloidSolid, geo.createCube, geo.createCone,
		geo.createConeSolid, geo.createBox, geo.createHyperbolicParaboloid,
		geo.createCylinder, geo.createCylinderSolid, geo.createPotentialWell,
		geo.createCardioidRevolution, geo.createCardioidRevolutionSolid,
		geo.createLemniscateRevolution, geo.createLemniscateRevolutionSolid,
		geo.createRose4LeafRevolution, geo.createRose4LeafRevolutionSolid,
	}

	// create geometric references dimension file
	fdim, err := os.Create(filepath.Join(dataDir, geometricrefdims))
	if err != nil {
		fmt.Printf("create file %s error: %v\n", geometricrefdims, err.Error())
		return fmt.Errorf("create file %s error: %v", geometricrefdims, err.Error())
	}
	defer fdim.Close()
	// loop over classes
	for i, class := range geometricObjects {
		//   create class specific file for the mass sums in each axis, plane, and row/column
		fclass, err := os.Create(filepath.Join(dataDir, class+".txt"))
		if err != nil {
			fmt.Printf("create file %s error: %v\n", class+".txt", err.Error())
			return fmt.Errorf("create file %s error: %v", class+".txt", err.Error())
		}

		// clear the previous densities
		for j := range geo.density {
			for k := range geo.density[j] {
				for m := range geo.density[j][k] {
					geo.density[j][k][m] = 0
				}
			}
		}

		// construct this class but don't save to disk
		geofunc[i]()

		var (
			rowFirst = planeDim
			rowLast  = 0
			colFirst = planeDim
			colLast  = 0
			sum      = 0
		)

		// ---------------------- axes = i ------------------------
		// loop over planes
		// For each plane, find the extent of the density
		// in the two axes, then find the mass for the rows and columns
		// inside these bounds.
	planeLoopI:
		for plane := range planeDim {
			// find the row bounds
			// loop over rows from first to last
			rowFirst = planeDim
			for row := range planeDim {
				sum = 0
				for col := range planeDim {
					// sum the row densities, if non-zero, save this row number, break
					sum += int(geo.density[plane][row][col])
				}
				if sum > 0 {
					rowFirst = row
					break
				}
			}

			// check if mass in this plane, if no mass go to next plane
			if rowFirst == planeDim {
				nrows := 0
				ncols := 0
				// write nrows and ncols to geometricrefdim on one line with space between
				fmt.Fprintf(fdim, "%d %d\n", nrows, ncols)
				continue planeLoopI
			}

			// loop over rows from last to first
			for row := planeDim - 1; row >= 0; row-- {
				sum = 0
				for col := range planeDim {
					// sum the densities, if non-zero, save this row number, break
					sum += int(geo.density[plane][row][col])
				}
				if sum > 0 {
					rowLast = row
					break
				}
			}

			// find the column bounds
			// loop over cols from first to last
			for col := range planeDim {
				sum = 0
				for row := range planeDim {
					//sum the col densities, if non-zero, save this col number, break
					sum += int(geo.density[plane][row][col])
				}
				if sum > 0 {
					colFirst = col
					break
				}
			}

			// loop over cols from last to first
			for col := planeDim - 1; col >= 0; col-- {
				sum = 0
				for row := range planeDim {
					//sum the col densities, if non-zero, save this col number, break
					sum += int(geo.density[plane][row][col])
				}
				if sum > 0 {
					colLast = col
					break
				}
			}

			nrows := rowLast - rowFirst + 1
			ncols := colLast - colFirst + 1
			// write nrows and ncols to geometricrefdim on one line with space between
			fmt.Fprintf(fdim, "%d %d\n", nrows, ncols)

			// loop from first non-zero row sum to last row non-zero sum
			for row := rowFirst; row < rowLast; row++ {
				sum := 0
				for col := range planeDim {
					// sum each row density and write to file geometricrefmass on the same line with a space between
					sum += int(geo.density[plane][row][col])
				}
				fmt.Fprintf(fclass, "%d ", sum)
			}
			sum = 0
			for col := range planeDim {
				// sum each row density and write to file geometricrefmass on the same line with a space between
				sum += int(geo.density[plane][rowLast][col])
			}
			fmt.Fprintf(fclass, "%d\n", sum)

			// loop from first non-zero col sum to last non-zero col sum
			for col := colFirst; col < colLast; col++ {
				sum = 0
				for row := range planeDim {
					// sum each col density and write to file geometricrefmass on the same line with a space between
					sum += int(geo.density[plane][row][col])
				}
				fmt.Fprintf(fclass, "%d ", sum)
			}
			sum = 0
			for row := range planeDim {
				// sum each col density and write to file geometricrefmass on the same line with a space between
				sum += int(geo.density[plane][row][colLast])
			}
			fmt.Fprintf(fclass, "%d\n", sum)
		}

		// -------------------------- axes = j -----------------------------
		// loop over planes
		// For each plane, find the extent of the density
		// in the two axes, then find the mass for the rows and columns
		// inside these bounds.
	planeLoopJ:
		for plane := range planeDim {
			// find the row bounds
			// loop over rows from first to last
			rowFirst = planeDim
			for row := range planeDim {
				sum = 0
				for col := range planeDim {
					// sum the row densities, if non-zero, save this row number, break
					sum += int(geo.density[row][plane][col])
				}
				if sum > 0 {
					rowFirst = row
					break
				}
			}

			// check if mass in this plane, if no mass go to next plane
			if rowFirst == planeDim {
				nrows := 0
				ncols := 0
				// write nrows and ncols to geometricrefdim on one line with space between
				fmt.Fprintf(fdim, "%d %d\n", nrows, ncols)
				continue planeLoopJ
			}

			// loop over rows from last to first
			for row := planeDim - 1; row >= 0; row-- {
				sum = 0
				for col := range planeDim {
					// sum the densities, if non-zero, save this row number, break
					sum += int(geo.density[row][plane][col])
				}
				if sum > 0 {
					rowLast = row
					break
				}
			}

			// find the column bounds
			// loop over cols from first to last
			for col := range planeDim {
				sum = 0
				for row := range planeDim {
					//sum the col densities, if non-zero, save this col number, break
					sum += int(geo.density[row][plane][col])
				}
				if sum > 0 {
					colFirst = col
					break
				}
			}

			// loop over cols from last to first
			for col := planeDim - 1; col >= 0; col-- {
				sum = 0
				for row := range planeDim {
					//sum the col densities, if non-zero, save this col number, break
					sum += int(geo.density[row][plane][col])
				}
				if sum > 0 {
					colLast = col
					break
				}
			}

			nrows := rowLast - rowFirst + 1
			ncols := colLast - colFirst + 1
			// write nrows and ncols to geometricrefdim on one line with space between
			fmt.Fprintf(fdim, "%d %d\n", nrows, ncols)

			// loop from first non-zero row sum to last row non-zero sum
			for row := rowFirst; row < rowLast; row++ {
				sum := 0
				for col := range planeDim {
					// sum each row density and write to file geometricrefmass on the same line with a space between
					sum += int(geo.density[row][plane][col])
				}
				fmt.Fprintf(fclass, "%d ", sum)
			}
			sum = 0
			for col := range planeDim {
				// sum each row density and write to file geometricrefmass on the same line with a space between
				sum += int(geo.density[rowLast][plane][col])
			}
			fmt.Fprintf(fclass, "%d\n", sum)

			// loop from first non-zero col sum to last non-zeron col sum
			for col := colFirst; col < colLast; col++ {
				sum = 0
				for row := range planeDim {
					// sum each col density and write to file geometricrefmass on the same line with a space between
					sum += int(geo.density[row][plane][col])
				}
				fmt.Fprintf(fclass, "%d ", sum)
			}
			sum = 0
			for row := range planeDim {
				// sum each col density and write to file geometricrefmass on the same line with a space between
				sum += int(geo.density[row][plane][colLast])
			}
			fmt.Fprintf(fclass, "%d\n", sum)
		}

		// -------------------------- axes = k --------------------------------------
		// loop over planes
		// For each plane, find the extent of the density
		// in the two axes, then find the mass for the rows and columns
		// inside these bounds.
	planeLoopK:
		for plane := range planeDim {
			// find the row bounds
			// loop over rows from first to last
			rowFirst = planeDim
			for row := range planeDim {
				sum = 0
				for col := range planeDim {
					// sum the row densities, if non-zero, save this row number, break
					sum += int(geo.density[row][col][plane])
				}
				if sum > 0 {
					rowFirst = row
					break
				}
			}

			// check if mass in this plane, if no mass go to next plane
			if rowFirst == planeDim {
				nrows := 0
				ncols := 0
				// write nrows and ncols to geometricrefdim on one line with space between
				fmt.Fprintf(fdim, "%d %d\n", nrows, ncols)
				continue planeLoopK
			}

			// loop over rows from last to first
			for row := planeDim - 1; row >= 0; row-- {
				sum = 0
				for col := range planeDim {
					// sum the densities, if non-zero, save this row number, break
					sum += int(geo.density[row][col][plane])
				}
				if sum > 0 {
					rowLast = row
					break
				}
			}

			// find the column bounds
			// loop over cols from first to last
			for col := range planeDim {
				sum = 0
				for row := range planeDim {
					//sum the col densities, if non-zero, save this col number, break
					sum += int(geo.density[row][col][plane])
				}
				if sum > 0 {
					colFirst = col
					break
				}
			}

			// loop over cols from last to first
			for col := planeDim - 1; col >= 0; col-- {
				sum = 0
				for row := range planeDim {
					//sum the col densities, if non-zero, save this col number, break
					sum += int(geo.density[row][col][plane])
				}
				if sum > 0 {
					colLast = col
					break
				}
			}

			nrows := rowLast - rowFirst + 1
			ncols := colLast - colFirst + 1
			// write nrows and ncols to geometricrefdim on one line with space between
			fmt.Fprintf(fdim, "%d %d\n", nrows, ncols)

			// loop from first non-zero row sum to last row non-zero sum
			for row := rowFirst; row < rowLast; row++ {
				sum := 0
				for col := range planeDim {
					// sum each row density and write to file geometricrefmass on the same line with a space between
					sum += int(geo.density[row][col][plane])
				}
				fmt.Fprintf(fclass, "%d ", sum)
			}
			sum = 0
			for col := range planeDim {
				// sum each row density and write to file geometricrefmass on the same line with a space between
				sum += int(geo.density[rowLast][col][plane])
			}
			fmt.Fprintf(fclass, "%d\n", sum)

			// loop from first non-zero col sum to last non-zeron col sum
			for col := colFirst; col < colLast; col++ {
				sum = 0
				for row := range planeDim {
					// sum each col density and write to file geometricrefmass on the same line with a space between
					sum += int(geo.density[row][col][plane])
				}
				fmt.Fprintf(fclass, "%d ", sum)
			}
			sum = 0
			for row := range planeDim {
				// sum each col density and write to file geometricrefmass on the same line with a space between
				sum += int(geo.density[row][colLast][plane])
			}
			fmt.Fprintf(fclass, "%d\n", sum)
		}
		err = fclass.Sync()
		if err != nil {
			fmt.Printf("Sync %s error: %v\n", class+".txt", err.Error())
		}
		err = fclass.Close()
		if err != nil {
			fmt.Printf("Close %s error: %v\n", class+".txt", err.Error())
		}
	}
	return nil
}

// create a geometric 3D object using its densities
func CreateObject(geometricObject string, noiseLevel int, shift bool) error {
	// create a GeoObject instance
	geo := GeoObject{
		density:    make([][][]byte, planeDim),
		noiseLevel: noiseLevel,
		shift:      shift,
	}
	for i := range geo.density {
		geo.density[i] = make([][]byte, planeDim)
		for j := range geo.density[i] {
			geo.density[i][j] = make([]byte, planeDim)
		}
	}

	// determine the geometric surface/solid
	switch geometricObject {
	case "ellipsoidsurface":
		geo.createEllipsoid()
	case "ellipsoidsolid":
		geo.createEllipsoidSolid()
	case "plane":
		geo.createPlane()
	case "paraboloid":
		geo.createParaboloid()
	case "paraboloidsolid":
		geo.createParaboloidSolid()
	case "cube":
		geo.createCube()
	case "cone":
		geo.createCone()
	case "conesolid":
		geo.createConeSolid()
	case "box":
		geo.createBox()
	case "hyperbolicparaboloid":
		geo.createHyperbolicParaboloid()
	case "cylindersurface":
		geo.createCylinder()
	case "cylindersolid":
		geo.createCylinderSolid()
	case "potentialwell":
		geo.createPotentialWell()
	case "cardioidrevolution":
		geo.createCardioidRevolution()
	case "cardioidrevolutionsolid":
		geo.createCardioidRevolutionSolid()
	case "lemniscaterevolution":
		geo.createLemniscateRevolution()
	case "lemniscaterevolutionsolid":
		geo.createLemniscateRevolutionSolid()
	case "rose4leafrevolution":
		geo.createRose4LeafRevolution()
	case "rose4leafrevolutionsolid":
		geo.createRose4LeafRevolutionSolid()
	case "geometricreferences":
		err := geo.createGeometricReferences()
		if err != nil {
			return fmt.Errorf("createGeometricReferences error: %v", err.Error())
		}
	default:
		fmt.Printf("create geometric object unknown case: '%s'\n", geometricObject)
		return fmt.Errorf("create geometric object unknown case %s", geometricObject)
	}
	
	// Save geometric object
	f, err := os.Create(filepath.Join(dataDir, geometricobject))
	if err != nil {
		fmt.Printf("create %s error: %v\n", geometricObject, err.Error())
		return fmt.Errorf("create %s error: %v", geometricObject, err.Error())
	}
	defer f.Close()

	for i := 0; i < planeDim; i++ {
		for j := 0; j < planeDim; j++ {
			for k := 0; k < planeDim-1; k++ {
				fmt.Fprintf(f, "%d ", geo.density[i][j][k])
			}
			fmt.Fprintf(f, "%d\n", geo.density[i][j][planeDim-1])
		}
	}

	return nil
}
