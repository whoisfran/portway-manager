//go:build !linux

package main

import "context"

// watchSystemTheme no hace falta fuera de Linux: WebView2 (Windows) y
// WKWebView (macOS) ya reevaluan @media (prefers-color-scheme) solos
// cuando el sistema cambia de tema con la app abierta (ver
// frontend/src/stores/theme.ts). Ver systemtheme_linux.go para el caso
// que si lo necesita.
func watchSystemTheme(_ context.Context) {}
