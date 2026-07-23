//go:build linux && cgo && gtk3

// Package webkitua overrides the WebKitGTK webview's User-Agent string.
//
// Wails v3's GTK3 Linux backend hardcodes the UA via
// webkit_settings_set_user_agent_with_application_details(settings, "wails.io", ""),
// which appends a non-standard token to the default WebKit UA string. Sites
// behind bot-detection (Cloudflare, etc. — e.g. claude.ai) fingerprint that
// as non-browser traffic and reject the request outright before any JS runs,
// so overriding the UA client-side via JS injection can't fix it: the
// request headers are already wrong by the time a page loads. This package
// reaches into the live GtkWindow to find its WebKitWebView and replace the
// UA with a real browser string instead.
package webkitua

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.1

#include <stdlib.h>
#include <gtk/gtk.h>
#include <webkit2/webkit2.h>

static WebKitWebView* find_webview(GtkWidget *widget) {
	if (widget == NULL) {
		return NULL;
	}
	if (WEBKIT_IS_WEB_VIEW(widget)) {
		return WEBKIT_WEB_VIEW(widget);
	}
	if (GTK_IS_CONTAINER(widget)) {
		WebKitWebView *found = NULL;
		GList *children = gtk_container_get_children(GTK_CONTAINER(widget));
		for (GList *iter = children; iter != NULL; iter = iter->next) {
			found = find_webview(GTK_WIDGET(iter->data));
			if (found != NULL) {
				break;
			}
		}
		g_list_free(children);
		return found;
	}
	return NULL;
}

static int set_user_agent(void *nativeWindow, const char *ua) {
	WebKitWebView *webview = find_webview(GTK_WIDGET(nativeWindow));
	if (webview == NULL) {
		return 0;
	}
	WebKitSettings *settings = webkit_web_view_get_settings(webview);
	webkit_settings_set_user_agent(settings, ua);
	return 1;
}
*/
import "C"
import "unsafe"

// SetUserAgent finds the WebKitWebView inside the given native GtkWindow
// pointer (as returned by wails' WebviewWindow.NativeWindow()) and sets its
// User-Agent. Returns false if no WebKitWebView could be found yet (e.g.
// called before the window's widgets are fully constructed).
func SetUserAgent(nativeWindow unsafe.Pointer, userAgent string) bool {
	if nativeWindow == nil {
		return false
	}
	cua := C.CString(userAgent)
	defer C.free(unsafe.Pointer(cua))
	return C.set_user_agent(nativeWindow, cua) != 0
}
