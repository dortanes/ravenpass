#ifndef RAVENPASS_LOGIN_ITEM_H
#define RAVENPASS_LOGIN_ITEM_H

#include <stdbool.h>

enum {
    RAVENPASS_LOGIN_ITEM_OFF,
    RAVENPASS_LOGIN_ITEM_ENABLED,
    RAVENPASS_LOGIN_ITEM_BLOCKED,
};

int ravenpass_login_item_status(void);
// Returns 0, or the system's error code when the item did not reach the wanted state.
long ravenpass_login_item_set(bool enabled);
void ravenpass_login_item_open_settings(void);
void ravenpass_login_item_watch_launch(void);
bool ravenpass_login_item_launched_at_login(void);

#endif
