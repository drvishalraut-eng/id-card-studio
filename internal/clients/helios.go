package clients

import (
	"fmt"
	"math"
	"strings"
)

// heliosSunburstPath reproduces the "rays" path computed by
// reference/index.html's inline script (the loop building the #rays "d"
// attribute), so the seeded Helios logo renders identically to the
// approved reference mockup.
func heliosSunburstPath() string {
	lengths := [5]float64{16, 9, 13, 6, 11}
	point := func(cx, sy, r float64) string {
		return fmt.Sprintf("%.1f %.1f", 80+r*cx, 80+r*sy)
	}

	var b strings.Builder
	for i := 0; i < 56; i++ {
		a := float64(i) * math.Pi / 28
		c, s := math.Cos(a), math.Sin(a)
		outer := 42 + lengths[i%5]*0.8
		crossEnd := 70.0
		if i%2 == 0 {
			crossEnd = 74
		}
		fmt.Fprintf(&b, "M%sL%sM%sL%s", point(c, s, 42), point(c, s, outer), point(c, s, 60), point(c, s, crossEnd))
	}
	return b.String()
}

// HeliosLogoSVG is "the built-in SVG from the reference" used to seed the
// Helios Material Handling client: the sunburst mark only (self-contained,
// viewBox "0 0 160 160"), reproducing reference/index.html's
// <symbol id="helios-mark"> standalone so it validates and renders on its
// own. The letter-spaced "HELIOS MATERIAL"/"HANDLING" wordmark next to it
// in the reference is plain styled HTML text, not part of the SVG — the
// card renderer (task 13) reproduces that lockup directly rather than
// treating Helios as a generic uploaded-logo image.
func HeliosLogoSVG() string {
	return fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 160 160">`+
			`<defs><linearGradient id="hg" x1="0" y1="0" x2="0" y2="1">`+
			`<stop offset="0" stop-color="#F2C66D"/><stop offset="1" stop-color="#E7757E"/>`+
			`</linearGradient></defs>`+
			`<path d="%s" fill="none" stroke="url(#hg)" stroke-width="2.6" stroke-linecap="round"/>`+
			`<path d="M71 67v26M89 67v26M71 80h18" fill="none" stroke="url(#hg)" stroke-width="4" stroke-linecap="round"/>`+
			`</svg>`,
		heliosSunburstPath(),
	)
}
