//go:build linux && cgo && gtk3

// Package webkitcookies enables persistent (disk-backed) cookie storage for
// the WebKitGTK webview.
//
// Wails v3's GTK3 Linux backend creates the webview with the default
// WebKitWebContext but never calls
// webkit_cookie_manager_set_persistent_storage on it, so WebKitCookieManager
// falls back to its default: cookies live only in memory and vanish the
// moment the process exits. Local storage and IndexedDB are unaffected
// (WebKitWebsiteDataManager persists those to disk on its own), which is why
// logged-in state disappears on relaunch while other site data survives.
// This package reaches into the live GtkWindow to find its WebKitWebView and
// switches its cookie manager to SQLite-backed persistent storage, using the
// same base data directory WebKit already picked for the rest of the site
// data.
package webkitcookies

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

// WebKitWebsiteDataManager only reports a base_data_directory when one was
// explicitly configured via webkit_website_data_manager_new(). The default
// manager wails uses computes its per-type directories (local storage,
// IndexedDB, HSTS, ...) lazily and internally, so the getter returns NULL
// even though data is in fact being written under
// g_get_user_data_dir()/<prgname> (visible from the existing localstorage/
// hsts-storage.sqlite files there). Mirror that same default so the cookie
// database lands in the same directory as everything else.
static int enable_persistent_cookies(void *nativeWindow) {
	WebKitWebView *webview = find_webview(GTK_WIDGET(nativeWindow));
	if (webview == NULL) {
		return 0;
	}
	WebKitWebContext *context = webkit_web_view_get_context(webview);
	WebKitWebsiteDataManager *dataManager = webkit_web_context_get_website_data_manager(context);
	const gchar *baseDir = webkit_website_data_manager_get_base_data_directory(dataManager);
	gchar *fallbackDir = NULL;
	if (baseDir == NULL) {
		fallbackDir = g_build_filename(g_get_user_data_dir(), g_get_prgname(), NULL);
		baseDir = fallbackDir;
	}
	g_mkdir_with_parents(baseDir, 0700);
	gchar *cookiePath = g_build_filename(baseDir, "cookies.sqlite", NULL);
	WebKitCookieManager *cookieManager = webkit_web_context_get_cookie_manager(context);
	webkit_cookie_manager_set_persistent_storage(cookieManager, cookiePath, WEBKIT_COOKIE_PERSISTENT_STORAGE_SQLITE);
	g_free(cookiePath);
	if (fallbackDir != NULL) {
		g_free(fallbackDir);
	}
	return 1;
}
*/
import "C"
import "unsafe"

// EnablePersistentStorage finds the WebKitWebView inside the given native
// GtkWindow pointer (as returned by wails' WebviewWindow.NativeWindow()) and
// switches its cookie manager to disk-backed SQLite storage. Returns false
// if no WebKitWebView could be found yet (e.g. called before the window's
// widgets are fully constructed).
func EnablePersistentStorage(nativeWindow unsafe.Pointer) bool {
	if nativeWindow == nil {
		return false
	}
	return C.enable_persistent_cookies(nativeWindow) != 0
}
