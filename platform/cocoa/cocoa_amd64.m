//go:build darwin && cocoa && !nogui

#import <Cocoa/Cocoa.h>
#import <QuartzCore/CAMetalLayer.h>

// LuxAppDelegate handles NSApplication lifecycle events.
@interface LuxAppDelegate : NSObject <NSApplicationDelegate>
@property (nonatomic, strong) NSWindow* window;
@property (nonatomic, strong) NSView* contentView;
@property (nonatomic, strong) CAMetalLayer* metalLayer;
@end

@implementation LuxAppDelegate
- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    [NSApp activateIgnoringOtherApps:YES];
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    return YES;
}
@end

// LuxView is a custom NSView with a CAMetalLayer for wgpu rendering.
@interface LuxView : NSView
@property (nonatomic, strong) CAMetalLayer* metalLayer;
@end

@implementation LuxView
- (BOOL)wantsUpdateLayer { return YES; }
- (CALayer*)makeBackingLayer {
    self.metalLayer = [CAMetalLayer layer];
    return self.metalLayer;
}
- (BOOL)acceptsFirstResponder { return YES; }
- (void)keyDown:(NSEvent *)event {}
- (void)keyUp:(NSEvent *)event {}
@end

// C API implementation for the Go cocoa package.

void* lux_cocoa_init(const char* title, int width, int height) {
    @autoreleasepool {
        [NSApplication sharedApplication];

        LuxAppDelegate* delegate = [[LuxAppDelegate alloc] init];
        [NSApp setDelegate:delegate];

        NSRect frame = NSMakeRect(100, 100, width, height);
        NSUInteger style = NSWindowStyleMaskTitled |
                          NSWindowStyleMaskClosable |
                          NSWindowStyleMaskResizable |
                          NSWindowStyleMaskMiniaturizable;

        NSWindow* window = [[NSWindow alloc] initWithContentRect:frame
                                                       styleMask:style
                                                         backing:NSBackingStoreBuffered
                                                           defer:NO];

        LuxView* view = [[LuxView alloc] initWithFrame:frame];
        view.wantsLayer = YES;
        [window setContentView:view];
        [window setTitle:[NSString stringWithUTF8String:title]];
        [window makeKeyAndOrderFront:nil];

        delegate.window = window;
        delegate.contentView = view;
        delegate.metalLayer = view.metalLayer;

        return (__bridge_retained void*)delegate;
    }
}

void lux_cocoa_run(void* handle) {
    @autoreleasepool {
        [NSApp run];
    }
}

void lux_cocoa_destroy(void* handle) {
    @autoreleasepool {
        LuxAppDelegate* delegate = (__bridge_transfer LuxAppDelegate*)handle;
        [delegate.window close];
        delegate.window = nil;
    }
}

void lux_cocoa_set_title(void* handle, const char* title) {
    @autoreleasepool {
        LuxAppDelegate* delegate = (__bridge LuxAppDelegate*)handle;
        [delegate.window setTitle:[NSString stringWithUTF8String:title]];
    }
}

void lux_cocoa_get_size(void* handle, int* width, int* height) {
    LuxAppDelegate* delegate = (__bridge LuxAppDelegate*)handle;
    NSRect frame = [delegate.window contentRectForFrameRect:delegate.window.frame];
    *width = (int)frame.size.width;
    *height = (int)frame.size.height;
}

void lux_cocoa_set_size(void* handle, int width, int height) {
    @autoreleasepool {
        LuxAppDelegate* delegate = (__bridge LuxAppDelegate*)handle;
        NSRect frame = delegate.window.frame;
        frame.size = NSMakeSize(width, height);
        [delegate.window setFrame:frame display:YES animate:YES];
    }
}

void lux_cocoa_set_fullscreen(void* handle, int fullscreen) {
    @autoreleasepool {
        LuxAppDelegate* delegate = (__bridge LuxAppDelegate*)handle;
        BOOL isFS = (delegate.window.styleMask & NSWindowStyleMaskFullScreen) != 0;
        if ((fullscreen && !isFS) || (!fullscreen && isFS)) {
            [delegate.window toggleFullScreen:nil];
        }
    }
}

void lux_cocoa_set_clipboard(const char* text) {
    @autoreleasepool {
        NSPasteboard* pb = [NSPasteboard generalPasteboard];
        [pb clearContents];
        [pb setString:[NSString stringWithUTF8String:text]
              forType:NSPasteboardTypeString];
    }
}

const char* lux_cocoa_get_clipboard(void) {
    @autoreleasepool {
        NSPasteboard* pb = [NSPasteboard generalPasteboard];
        NSString* str = [pb stringForType:NSPasteboardTypeString];
        if (str == nil) return NULL;
        return [str UTF8String];
    }
}

void* lux_cocoa_get_metal_layer(void* handle) {
    LuxAppDelegate* delegate = (__bridge LuxAppDelegate*)handle;
    return (__bridge void*)delegate.metalLayer;
}
