package clients

import "testing"

func TestValidateSVGAcceptsValidLogo(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 40"><path d="M0 0h100v40h-100z" fill="#fff"/></svg>`
	if err := ValidateSVG([]byte(svg)); err != nil {
		t.Fatalf("expected a valid SVG to pass, got %v", err)
	}
}

func TestValidateSVGRejectsNonSVG(t *testing.T) {
	if err := ValidateSVG([]byte("not xml at all")); err == nil {
		t.Fatal("expected an error for non-XML input")
	}
	if err := ValidateSVG([]byte(`<div>hello</div>`)); err == nil {
		t.Fatal("expected an error for a non-SVG root element")
	}
}

func TestValidateSVGRejectsMissingViewBox(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0h10v10z"/></svg>`
	if err := ValidateSVG([]byte(svg)); err == nil {
		t.Fatal("expected an error for a missing viewBox")
	}
}

func TestValidateSVGRejectsText(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><text x="0" y="0">hi</text></svg>`
	if err := ValidateSVG([]byte(svg)); err == nil {
		t.Fatal("expected an error for <text>")
	}
}

func TestValidateSVGRejectsImage(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><image href="logo.png"/></svg>`
	if err := ValidateSVG([]byte(svg)); err == nil {
		t.Fatal("expected an error for <image>")
	}
}

func TestValidateSVGRejectsScript(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><script>alert(1)</script></svg>`
	if err := ValidateSVG([]byte(svg)); err == nil {
		t.Fatal("expected an error for <script>")
	}
}

func TestValidateSVGRejectsForeignObject(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><foreignObject><div>hi</div></foreignObject></svg>`
	if err := ValidateSVG([]byte(svg)); err == nil {
		t.Fatal("expected an error for <foreignObject>")
	}
}

func TestValidateSVGRejectsEventHandlerAttribute(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><path onclick="alert(1)" d="M0 0h10v10z"/></svg>`
	if err := ValidateSVG([]byte(svg)); err == nil {
		t.Fatal("expected an error for an on* attribute")
	}
}

func TestValidateSVGRejectsExternalHref(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><use href="https://evil.example/x.svg#y"/></svg>`
	if err := ValidateSVG([]byte(svg)); err == nil {
		t.Fatal("expected an error for an external href")
	}
}

func TestValidateSVGAllowsLocalFragmentHref(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><defs><path id="p" d="M0 0h10v10z"/></defs><use href="#p"/></svg>`
	if err := ValidateSVG([]byte(svg)); err != nil {
		t.Fatalf("expected a local #fragment href to be allowed, got %v", err)
	}
}

func TestValidateSVGRejectsDoctype(t *testing.T) {
	svg := `<?xml version="1.0"?><!DOCTYPE svg [<!ENTITY x "y">]><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"/>`
	if err := ValidateSVG([]byte(svg)); err == nil {
		t.Fatal("expected an error for a DOCTYPE/ENTITY declaration")
	}
}
