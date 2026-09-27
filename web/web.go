// Package web embeds the frontend assets served by ID Card Studio.
package web

import "embed"

//go:embed assets
var Assets embed.FS
