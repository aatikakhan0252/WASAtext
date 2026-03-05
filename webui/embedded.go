package webui

import "embed"

// Content holds our static web server content.
// Note: "all:" prefix is required to include dotfiles (e.g. .gitkeep)
//
//go:embed all:dist
var Content embed.FS
