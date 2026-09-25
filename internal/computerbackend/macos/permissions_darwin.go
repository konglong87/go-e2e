//go:build darwin && cgo

package macos

/*
#cgo LDFLAGS: -framework CoreGraphics -framework ApplicationServices
#include <CoreGraphics/CoreGraphics.h>
#include <ApplicationServices/ApplicationServices.h>

static void requestHostPermissions(void) {
    if (__builtin_available(macOS 10.15, *)) {
        (void)CGRequestScreenCaptureAccess();
    }
    const void *key = kAXTrustedCheckOptionPrompt;
    const void *value = kCFBooleanTrue;
    const void *keys[] = {key};
    const void *values[] = {value};
    CFDictionaryRef options = CFDictionaryCreate(kCFAllocatorDefault, keys, values, 1,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    (void)AXIsProcessTrustedWithOptions(options);
    if (options != NULL) CFRelease(options);
}
*/
import "C"

// RequestHostPermissions asks macOS to attribute the prompt to the signed
// desktop host process rather than only to the nested helper.
func RequestHostPermissions() {
	C.requestHostPermissions()
}
