#import <Cocoa/Cocoa.h>

@interface Platform: NSObject

+ (void)windowDidBecomeKey:(NSNotification *)aNotification;

@end

@implementation Platform

+ (void)windowDidBecomeKey:(NSNotification *)aNotification
{
    NSWindow *aWindow = aNotification.object;
    [[aWindow standardWindowButton:NSWindowZoomButton] setHidden:YES];
    if (@available(macOS 10.14, *)) {
        aWindow.appearance = [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua];
    }
}

@end

void platform_darwin_setup(void)
{
    NSNotificationCenter *defaultCenter = [NSNotificationCenter defaultCenter];
    [defaultCenter addObserver:[Platform class]
        selector:@selector(windowDidBecomeKey:)
        name:NSWindowDidBecomeKeyNotification
        object:nil];
}

void platform_darwin_after_file_dialog(void)
{
    NSWindow *firstWindow = [[NSApp windows] firstObject];
    [firstWindow performSelectorOnMainThread:@selector(makeKeyAndOrderFront:) withObject:NSApp waitUntilDone:NO];
}
