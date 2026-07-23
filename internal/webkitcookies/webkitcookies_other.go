//go:build !(linux && cgo && gtk3)

package webkitcookies

import "unsafe"

// EnablePersistentStorage is a no-op on platforms/backends where cookie
// persistence is already handled natively (macOS/Windows webviews) or where
// we haven't wired up a native override (GTK4/webkitgtk-6.0 Linux backend).
func EnablePersistentStorage(nativeWindow unsafe.Pointer) bool {
	return false
}
