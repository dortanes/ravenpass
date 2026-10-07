//go:build darwin && cgo

// memset_s is declared only on request.
#define __STDC_WANT_LIB_EXT1__ 1

#import <AppKit/AppKit.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>
#import <Vision/Vision.h>
#include <stdlib.h>
#include <string.h>

#include "qrcodes_darwin.h"

static ravenpass_qr_status ravenpass_qr_read(VNImageRequestHandler *handler, char **texts, size_t *length) {
    VNDetectBarcodesRequest *request = [[VNDetectBarcodesRequest alloc] init];
    request.symbologies = @[ VNBarcodeSymbologyQR ];
    if (![handler performRequests:@[ request ] error:nil]) {
        return ravenpass_qr_unreadable;
    }
    NSMutableData *found = [NSMutableData data];
    for (VNBarcodeObservation *code in request.results) {
        NSData *text = [code.payloadStringValue dataUsingEncoding:NSUTF8StringEncoding];
        if (text.length == 0) {
            continue;
        }
        [found appendData:text];
        [found appendBytes:"" length:1];
    }
    *length = found.length;
    *texts = malloc(found.length > 0 ? found.length : 1);
    if (*texts == NULL) {
        return ravenpass_qr_failed;
    }
    memcpy(*texts, found.bytes, found.length);
    memset_s(found.mutableBytes, found.length, 0, found.length);
    return ravenpass_qr_ok;
}

ravenpass_qr_status ravenpass_qr_picture(const void *content, size_t size, char **texts, size_t *length) {
    @autoreleasepool {
        NSData *data = [NSData dataWithBytesNoCopy:(void *)content length:size freeWhenDone:NO];
        VNImageRequestHandler *handler = [[VNImageRequestHandler alloc] initWithData:data options:@{}];
        return ravenpass_qr_read(handler, texts, length);
    }
}

ravenpass_qr_status ravenpass_qr_clipboard(char **texts, size_t *length) {
    @autoreleasepool {
        NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
        // A file copied in Finder puts its icon on the clipboard beside its address; the picture is the file.
        NSURL *file = [pasteboard readObjectsForClasses:@[ NSURL.class ] options:@{
            NSPasteboardURLReadingFileURLsOnlyKey : @YES,
            NSPasteboardURLReadingContentsConformToTypesKey : @[ UTTypeImage.identifier ],
        }].firstObject;
        if (file != nil) {
            VNImageRequestHandler *handler = [[VNImageRequestHandler alloc] initWithURL:file options:@{}];
            return ravenpass_qr_read(handler, texts, length);
        }
        NSImage *image = [pasteboard readObjectsForClasses:@[ NSImage.class ] options:nil].firstObject;
        CGImageRef picture = [image CGImageForProposedRect:NULL context:nil hints:nil];
        if (picture == NULL) {
            return ravenpass_qr_no_picture;
        }
        VNImageRequestHandler *handler = [[VNImageRequestHandler alloc] initWithCGImage:picture options:@{}];
        return ravenpass_qr_read(handler, texts, length);
    }
}

void ravenpass_qr_release(char *texts, size_t length) {
    memset_s(texts, length, 0, length);
    free(texts);
}
