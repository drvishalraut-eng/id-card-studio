package clients

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// disallowedElements mirrors the client-side SVG check: none of these may
// appear anywhere in an uploaded logo.
var disallowedElements = map[string]bool{
	"text":          true,
	"image":         true,
	"script":        true,
	"foreignobject": true,
}

// ValidateSVG rejects a logo that:
//   - is not valid SVG/XML, or has no root <svg> element
//   - has no viewBox on the root element
//   - contains <text>, <image>, <script> or <foreignObject>
//   - contains an on* event-handler attribute
//   - references anything outside the file (an href that isn't a local
//     "#fragment")
func ValidateSVG(data []byte) error {
	upper := strings.ToUpper(string(data))
	if strings.Contains(upper, "<!DOCTYPE") || strings.Contains(upper, "<!ENTITY") {
		return errors.New("SVG must not declare a DOCTYPE or ENTITY")
	}

	dec := xml.NewDecoder(strings.NewReader(string(data)))
	sawRoot := false
	hasViewBox := false
	depth := 0

	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("file is not valid SVG: %w", err)
		}

		switch el := tok.(type) {
		case xml.StartElement:
			depth++
			name := strings.ToLower(el.Name.Local)
			if depth == 1 {
				if name != "svg" {
					return errors.New("file is not an SVG (root element is not <svg>)")
				}
				sawRoot = true
			}
			if disallowedElements[name] {
				return fmt.Errorf("SVG must not contain <%s>", el.Name.Local)
			}
			for _, attr := range el.Attr {
				attrName := strings.ToLower(attr.Name.Local)
				if depth == 1 && attrName == "viewbox" {
					hasViewBox = true
				}
				if strings.HasPrefix(attrName, "on") {
					return fmt.Errorf("SVG must not contain event-handler attributes (found %q)", attr.Name.Local)
				}
				if attrName == "href" || strings.HasSuffix(attrName, ":href") {
					if attr.Value != "" && !strings.HasPrefix(attr.Value, "#") {
						return errors.New("SVG must not reference external files (found an href that is not a local #fragment)")
					}
				}
			}
		case xml.EndElement:
			depth--
		}
	}

	if !sawRoot {
		return errors.New("file is not an SVG (no root <svg> element found)")
	}
	if !hasViewBox {
		return errors.New("SVG must have a viewBox attribute")
	}
	return nil
}
