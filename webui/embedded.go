package webui

import "embed"

// Content holds our static web server content.
//
//go:embed dist/*
var Content embed.FS
