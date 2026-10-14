//go:build darwin

#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#include <dispatch/dispatch.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>
#include "computer_pointer_darwin.h"

static CGFloat const kPointerWidth = 320.0;
static CGFloat const kPointerHeight = 88.0;
static CGFloat const kPointerMargin = 12.0;

@class Goe2EComputerPointerController;

struct goe2e_computer_pointer {
    pthread_mutex_t mutex;
    bool closed;
    Goe2EComputerPointerController *controller;
};

@interface Goe2EComputerPointerView : NSView
@property(nonatomic, copy) NSString *phase;
@property(nonatomic, copy) NSString *appName;
@property(nonatomic, assign) BOOL visible;
@end

@interface Goe2EComputerPointerPanel : NSPanel
@end

@interface Goe2EComputerPointerController : NSObject
@property(nonatomic, assign) goe2e_computer_pointer *bridge;
@property(nonatomic, retain) Goe2EComputerPointerPanel *panel;
@property(nonatomic, retain) Goe2EComputerPointerView *view;
- (instancetype)initWithBridge:(goe2e_computer_pointer *)bridge;
- (void)showJSON:(char *)json;
- (void)hide;
- (void)close;
@end

@implementation Goe2EComputerPointerPanel
- (BOOL)canBecomeKeyWindow { return NO; }
- (BOOL)canBecomeMainWindow { return NO; }
@end

@implementation Goe2EComputerPointerView
- (BOOL)isOpaque { return NO; }
- (void)drawRect:(NSRect)dirtyRect {
    (void)dirtyRect;
    if (!self.visible) return;

    NSPoint tip = NSMakePoint(22.0, 23.0);
    NSBezierPath *arrow = [NSBezierPath bezierPath];
    [arrow moveToPoint:tip];
    [arrow lineToPoint:NSMakePoint(10.0, 62.0)];
    [arrow lineToPoint:NSMakePoint(28.0, 48.0)];
    [arrow lineToPoint:NSMakePoint(42.0, 67.0)];
    [arrow lineToPoint:NSMakePoint(50.0, 62.0)];
    [arrow lineToPoint:NSMakePoint(36.0, 43.0)];
    [arrow lineToPoint:NSMakePoint(57.0, 42.0)];
    [arrow closePath];
    [[NSColor colorWithWhite:0 alpha:0.88] setFill];
    [arrow fill];
    [[NSColor whiteColor] setStroke];
    [arrow setLineWidth:1.5];
    [arrow stroke];

    BOOL unknown = [self.phase isEqualToString:@"unknown"];
    NSColor *accent = unknown ? [NSColor colorWithCalibratedRed:0.98 green:0.56 blue:0.12 alpha:0.95]
                              : [NSColor colorWithCalibratedRed:0.22 green:0.78 blue:1.0 alpha:0.95];
    [accent setStroke];
    NSBezierPath *ring = [NSBezierPath bezierPathWithOvalInRect:NSMakeRect(8.0, 9.0, 28.0, 28.0)];
    [ring setLineWidth:2.5];
    [ring stroke];

    NSRect pill = NSMakeRect(54.0, 30.0, self.bounds.size.width - 66.0, 34.0);
    NSBezierPath *background = [NSBezierPath bezierPathWithRoundedRect:pill xRadius:12.0 yRadius:12.0];
    [[NSColor colorWithCalibratedWhite:0.06 alpha:0.92] setFill];
    [background fill];
    [accent setStroke];
    [background setLineWidth:1.0];
    [background stroke];

    NSString *prefix = unknown ? @"AI 已尝试操作" : @"AI 正在操作";
    NSString *app = self.appName.length > 0 ? self.appName : @"目标 App";
    NSString *text = [NSString stringWithFormat:@"%@ · %@", prefix, app];
    NSDictionary *attributes = @{
        NSFontAttributeName: [NSFont systemFontOfSize:12.0 weight:NSFontWeightSemibold],
        NSForegroundColorAttributeName: [NSColor whiteColor]
    };
    [text drawInRect:NSMakeRect(pill.origin.x + 12.0, pill.origin.y + 10.0, pill.size.width - 24.0, 16.0)
      withAttributes:attributes];
}
@end

static void OnPointerMainThread(void (^block)(void)) {
    // Never synchronously hop from the Go/runtime worker to AppKit's main
    // queue. The standalone runtime has no Wails main-loop ownership, and a
    // sync hop there can deadlock the action/runtime process. Ordering is
    // preserved by the serial main queue; lifecycle methods are idempotent.
    dispatch_async(dispatch_get_main_queue(), block);
}

static NSString *PointerAppName(NSString *bundleID) {
    if (bundleID.length == 0) return @"目标 App";
    NSRunningApplication *running = [NSRunningApplication runningApplicationsWithBundleIdentifier:bundleID].firstObject;
    if (running.localizedName.length > 0) return running.localizedName;
    NSURL *url = [NSWorkspace.sharedWorkspace URLForApplicationWithBundleIdentifier:bundleID];
    if (url.path.length > 0) {
        NSString *name = [[NSFileManager defaultManager] displayNameAtPath:url.path];
        if (name.length > 0) return name;
    }
    NSArray<NSString *> *parts = [bundleID componentsSeparatedByString:@"."];
    return parts.lastObject.length > 0 ? parts.lastObject : @"目标 App";
}

static CGFloat PointerPrimaryTop(void) {
    NSScreen *primary = NSScreen.screens.firstObject;
    if (primary != nil) return NSMaxY(primary.frame);
    NSScreen *main = NSScreen.mainScreen;
    return main != nil ? NSMaxY(main.frame) : 0.0;
}

static NSPoint PointerAppKitPointForGlobalPoint(NSPoint globalPoint) {
    // Native computer-use points are CG global top-left coordinates. AppKit
    // window frames use a global bottom-left origin, so flip only Y about the
    // primary screen's top edge. Do not flip about the target screen/window: a
    // secondary display may have a negative origin or a different height.
    return NSMakePoint(globalPoint.x, PointerPrimaryTop() - globalPoint.y);
}

static NSScreen *PointerScreenForPoint(NSPoint point) {
    for (NSScreen *screen in NSScreen.screens) {
        if (NSPointInRect(point, screen.frame)) return screen;
    }
    return NSScreen.mainScreen;
}

@implementation Goe2EComputerPointerController
- (instancetype)initWithBridge:(goe2e_computer_pointer *)bridge {
    self = [super init];
    if (!self) return nil;
    _bridge = bridge;
    _view = [[Goe2EComputerPointerView alloc] initWithFrame:NSMakeRect(0, 0, kPointerWidth, kPointerHeight)];
    _view.visible = NO;
    _panel = [[Goe2EComputerPointerPanel alloc] initWithContentRect:NSMakeRect(0, 0, kPointerWidth, kPointerHeight)
        styleMask:(NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel)
        backing:NSBackingStoreBuffered defer:NO];
    _panel.contentView = _view;
    _panel.opaque = NO;
    _panel.backgroundColor = NSColor.clearColor;
    _panel.hasShadow = NO;
    _panel.ignoresMouseEvents = YES;
    _panel.hidesOnDeactivate = NO;
    _panel.releasedWhenClosed = NO;
    _panel.animationBehavior = NSWindowAnimationBehaviorNone;
    _panel.level = NSStatusWindowLevel;
    _panel.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces |
        NSWindowCollectionBehaviorFullScreenAuxiliary |
        NSWindowCollectionBehaviorStationary;
    if ([_panel respondsToSelector:@selector(setSharingType:)]) {
        _panel.sharingType = NSWindowSharingNone;
    }
    return self;
}

- (void)showJSON:(char *)json {
    if (json == NULL) return;
    NSData *data = [NSData dataWithBytes:json length:strlen(json)];
    NSError *error = nil;
    NSDictionary *payload = [NSJSONSerialization JSONObjectWithData:data options:0 error:&error];
    if (error != nil || ![payload isKindOfClass:[NSDictionary class]]) return;
    NSDictionary *cue = payload[@"cue"];
    NSDictionary *point = [cue isKindOfClass:[NSDictionary class]] ? cue[@"global_point"] : nil;
    NSNumber *x = [point isKindOfClass:[NSDictionary class]] ? point[@"x"] : nil;
    NSNumber *y = [point isKindOfClass:[NSDictionary class]] ? point[@"y"] : nil;
    if (![x isKindOfClass:[NSNumber class]] || ![y isKindOfClass:[NSNumber class]]) return;

    NSString *phase = [cue[@"phase"] isKindOfClass:[NSString class]] ? cue[@"phase"] : @"dispatched";
    NSString *bundleID = [cue[@"app_bundle_id"] isKindOfClass:[NSString class]] ? cue[@"app_bundle_id"] : @"";
    self.view.phase = phase;
    self.view.appName = PointerAppName(bundleID);
    self.view.visible = YES;

    NSPoint globalPoint = NSMakePoint(x.doubleValue, y.doubleValue);
    NSPoint target = PointerAppKitPointForGlobalPoint(globalPoint);
    NSRect frame = NSMakeRect(target.x - 12.0, target.y - 12.0, kPointerWidth, kPointerHeight);
    NSScreen *screen = PointerScreenForPoint(target);
    if (screen != nil) {
        NSRect visible = screen.visibleFrame;
        frame.origin.x = MIN(MAX(NSMinX(visible) + kPointerMargin, frame.origin.x), NSMaxX(visible) - frame.size.width - kPointerMargin);
        frame.origin.y = MIN(MAX(NSMinY(visible) + kPointerMargin, frame.origin.y), NSMaxY(visible) - frame.size.height - kPointerMargin);
    }
    [self.panel setFrame:frame display:YES];
    [self.panel orderFrontRegardless];
    [self.view setNeedsDisplay:YES];
}

- (void)hide {
    self.view.visible = NO;
    [self.panel orderOut:nil];
}

- (void)close {
    [self hide];
    [self.panel close];
}

- (void)dealloc {
    [self close];
    [_view release];
    [_panel release];
    [super dealloc];
}
@end

goe2e_computer_pointer *goe2e_computer_pointer_create(void) {
    goe2e_computer_pointer *bridge = calloc(1, sizeof(goe2e_computer_pointer));
    if (bridge == NULL) return NULL;
    pthread_mutex_init(&bridge->mutex, NULL);
    OnPointerMainThread(^{
        bridge->controller = [[Goe2EComputerPointerController alloc] initWithBridge:bridge];
    });
    return bridge;
}

void goe2e_computer_pointer_show(goe2e_computer_pointer *bridge, const char *jsonUTF8) {
    if (bridge == NULL || jsonUTF8 == NULL) return;
    char *copy = strdup(jsonUTF8);
    if (copy == NULL) return;
    OnPointerMainThread(^{
        pthread_mutex_lock(&bridge->mutex);
        BOOL closed = bridge->closed;
        Goe2EComputerPointerController *controller = [bridge->controller retain];
        pthread_mutex_unlock(&bridge->mutex);
        if (!closed && controller != nil) [controller showJSON:copy];
        [controller release];
        free(copy);
    });
}

void goe2e_computer_pointer_hide(goe2e_computer_pointer *bridge) {
    if (bridge == NULL) return;
    OnPointerMainThread(^{
        pthread_mutex_lock(&bridge->mutex);
        BOOL closed = bridge->closed;
        Goe2EComputerPointerController *controller = [bridge->controller retain];
        pthread_mutex_unlock(&bridge->mutex);
        if (!closed && controller != nil) [controller hide];
        [controller release];
    });
}

void goe2e_computer_pointer_close(goe2e_computer_pointer *bridge) {
    if (bridge == NULL) return;
    pthread_mutex_lock(&bridge->mutex);
    if (bridge->closed) {
        pthread_mutex_unlock(&bridge->mutex);
        return;
    }
    bridge->closed = true;
    pthread_mutex_unlock(&bridge->mutex);
    OnPointerMainThread(^{
        pthread_mutex_lock(&bridge->mutex);
        Goe2EComputerPointerController *controller = [bridge->controller retain];
        bridge->controller = nil;
        pthread_mutex_unlock(&bridge->mutex);
        [controller close];
        [controller release];
        pthread_mutex_destroy(&bridge->mutex);
        free(bridge);
    });
}
