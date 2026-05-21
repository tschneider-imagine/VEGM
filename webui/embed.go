package webui

import "embed"

// StaticFS contains the browser UI assets served by VEGM.
//go:embed static/*
var StaticFS embed.FS
