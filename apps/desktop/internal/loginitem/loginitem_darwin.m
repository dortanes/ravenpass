//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <CoreServices/CoreServices.h>
#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>

#include "loginitem_darwin.h"

// Read and written on the main thread.
static bool ravenpass_launched_at_login = false;

int ravenpass_login_item_status(void) {
    @autoreleasepool {
        switch (SMAppService.mainAppService.status) {
        case SMAppServiceStatusEnabled:
            return RAVENPASS_LOGIN_ITEM_ENABLED;
        case SMAppServiceStatusRequiresApproval:
            return RAVENPASS_LOGIN_ITEM_BLOCKED;
        default:
            return RAVENPASS_LOGIN_ITEM_OFF;
        }
    }
}

long ravenpass_login_item_set(bool enabled) {
    @autoreleasepool {
        SMAppService *item = SMAppService.mainAppService;
        NSError *error = nil;
        BOOL changed = enabled ? [item registerAndReturnError:&error] : [item unregisterAndReturnError:&error];
        if (changed) {
            return 0;
        }
        // These calls also fail for an item already in the wanted state and for one the owner blocked.
        SMAppServiceStatus status = item.status;
        BOOL reached = enabled
            ? status == SMAppServiceStatusEnabled || status == SMAppServiceStatusRequiresApproval
            : status == SMAppServiceStatusNotRegistered;
        return reached ? 0 : (error.code != 0 ? (long)error.code : -1);
    }
}

void ravenpass_login_item_open_settings(void) {
    [SMAppService openSystemSettingsLoginItems];
}

// The open-application event is current only while the launch notification is delivered, which Wails forwards to Go too late.
void ravenpass_login_item_watch_launch(void) {
    [NSNotificationCenter.defaultCenter addObserverForName:NSApplicationDidFinishLaunchingNotification
                                                    object:nil
                                                     queue:nil
                                                usingBlock:^(NSNotification *notification) {
        NSAppleEventDescriptor *event = NSAppleEventManager.sharedAppleEventManager.currentAppleEvent;
        ravenpass_launched_at_login = event.eventClass == kCoreEventClass && event.eventID == kAEOpenApplication &&
            [event paramDescriptorForKeyword:keyAEPropData].enumCodeValue == keyAELaunchedAsLogInItem;
    }];
}

// Waiting for the main queue lets a notification still being delivered reach the observer first.
bool ravenpass_login_item_launched_at_login(void) {
    __block bool launched = false;
    dispatch_block_t read = ^{
        launched = ravenpass_launched_at_login;
    };
    if ([NSThread isMainThread]) {
        read();
    } else {
        dispatch_sync(dispatch_get_main_queue(), read);
    }
    return launched;
}
