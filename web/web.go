// Package web embeds the frontend served by ID Card Studio: the app shell
// (index.html), its JS/CSS (static/), brand assets and the Excel template
// (assets/), and the pinned third-party libraries and fonts (vendor/).
package web

import "embed"

//go:embed index.html static assets vendor
var FS embed.FS
