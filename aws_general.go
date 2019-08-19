package lblconv

// AWSBoundingBox defines an axis-aligned rectangle with the dimensions given as normalised ratios
// of the image size.
type AWSBoundingBox struct {
	Left   float64
	Top    float64
	Width  float64
	Height float64
}

// AWSPoint defines a point in an image. The coordinates are normalised ratios of the image size.
type AWSPoint struct {
	X float64
	Y float64
}
