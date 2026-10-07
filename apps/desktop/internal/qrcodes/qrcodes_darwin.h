#ifndef RAVENPASS_QRCODES_H
#define RAVENPASS_QRCODES_H

#include <stddef.h>

typedef enum {
    ravenpass_qr_ok,
    ravenpass_qr_no_picture,
    ravenpass_qr_unreadable,
    ravenpass_qr_failed,
} ravenpass_qr_status;

// On ravenpass_qr_ok, texts holds each QR code's UTF-8 text followed by NUL; ravenpass_qr_release frees it.
ravenpass_qr_status ravenpass_qr_picture(const void *content, size_t size, char **texts, size_t *length);
ravenpass_qr_status ravenpass_qr_clipboard(char **texts, size_t *length);
void ravenpass_qr_release(char *texts, size_t length);

#endif
