//go:build !(linux && cgo && gtk3)

package webkitua

import "unsafe"

// SetUserAgent is a no-op on platforms/backends where we haven't wired up a
// native override (macOS/Windows webviews already send normal browser UAs;
// the GTK4/webkitgtk-6.0 Linux backend isn't targeted by this build).
func SetUserAgent(nativeWindow unsafe.Pointer, userAgent string) bool {
	return false
}
