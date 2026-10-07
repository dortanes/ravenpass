package com.dortanes.ravenpass.autofill;

import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.content.IntentSender;
import android.graphics.drawable.Icon;
import android.os.Build;
import android.os.Bundle;
import android.service.autofill.Dataset;
import android.service.autofill.Field;
import android.service.autofill.FillResponse;
import android.service.autofill.InlinePresentation;
import android.service.autofill.Presentations;
import android.service.autofill.SaveInfo;
import android.view.autofill.AutofillId;
import android.view.autofill.AutofillValue;
import android.view.inputmethod.InlineSuggestionsRequest;
import android.widget.RemoteViews;

import androidx.annotation.RequiresApi;

import com.wails.app.R;

import org.json.JSONArray;
import org.json.JSONObject;

import java.util.Base64;
import java.util.Objects;
import java.util.UUID;
import java.util.stream.Stream;

/** Values never travel in a suggestion: each authenticates through the fill screen, which asks the core. */
final class Responses {
    // The client state a save request carries back: who asked, and the fields of the sign-in.
    static final String REQUESTER = "com.dortanes.ravenpass.autofill.REQUESTER";
    static final String SAVE_USERNAME = "com.dortanes.ravenpass.autofill.SAVE_USERNAME";
    static final String SAVE_PASSWORD = "com.dortanes.ravenpass.autofill.SAVE_PASSWORD";
    // The extras of the fill screen's intent: the chosen credential and the label it shows by.
    static final String CREDENTIAL = "com.dortanes.ravenpass.autofill.CREDENTIAL";
    static final String LABEL = "com.dortanes.ravenpass.autofill.LABEL";

    private Responses() {
    }

    /** null where nothing is offered. */
    static FillResponse fill(Context context, Screen screen, JSONObject answer, InlineSuggestionsRequest inline) {
        JSONObject form = answer.optJSONObject("form");
        if (form == null) {
            return null;
        }
        Presenting presenting = Presenting.of(context, inline);
        switch (Core.status(answer)) {
            case Core.LOCKED:
                return locked(context, screen, form, presenting);
            case Core.OK:
                return open(context, screen, answer, form, presenting);
            default:
                return null;
        }
    }

    /** The account a suggestion names, then the tags that tell it apart from others on the site. */
    private static String withTags(String account, JSONArray tags) {
        if (tags == null) {
            return account;
        }
        StringBuilder line = new StringBuilder(account);
        for (int i = 0; i < tags.length(); i++) {
            String tag = tags.optString(i);
            if (tag.isEmpty()) {
                continue;
            }
            if (line.length() > 0) {
                line.append(" · ");
            }
            line.append(tag);
        }
        return line.toString();
    }

    /** Tells nothing about the vault; the unlock answers with the open vault's entries. */
    private static FillResponse locked(Context context, Screen screen, JSONObject form, Presenting presenting) {
        AutofillId[] fields = fields(screen, form);
        if (fields.length == 0) {
            return null;
        }
        CharSequence action = context.getString(R.string.autofill_entry_unlock);
        IntentSender unlock = sender(context, new Intent(context, UnlockActivity.class), true);
        RemoteViews menu = Presenting.entryMenu(context, action);
        InlinePresentation chip = presenting.entry(0, action, context.getString(R.string.autofill_unlock));
        FillResponse.Builder response = new FillResponse.Builder();
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            response.setAuthentication(fields, unlock, presentations(menu, chip));
        } else {
            authenticate(response, fields, unlock, menu, chip);
        }
        return response.build();
    }

    private static FillResponse open(Context context, Screen screen, JSONObject answer, JSONObject form,
            Presenting presenting) {
        Destination destination = Destination.of(answer, screen);
        if (destination == null) {
            return null;
        }
        FillResponse.Builder response = new FillResponse.Builder();
        boolean offered = false;
        AutofillId[] fields = destination.fields();
        if (fields.length > 0) {
            boolean search = answer.optBoolean("search");
            JSONArray suggestions = answer.optJSONArray("suggestions");
            JSONObject icons = answer.optJSONObject("icons");
            int count = suggestions != null ? suggestions.length() : 0;
            int chips = Math.max(0, presenting.chips() - (search ? 1 : 0));
            for (int i = 0; i < count; i++) {
                JSONObject suggestion = suggestions.optJSONObject(i);
                String label = suggestion.optString("label");
                String account = withTags(suggestion.optString("account"), suggestion.optJSONArray("tags"));
                Icon face = presenting.face(label, siteIcon(icons, suggestion.optString("site")));
                Intent fill = destination.write(new Intent(context, FillActivity.class))
                        .putExtra(CREDENTIAL, suggestion.optString("id"))
                        .putExtra(LABEL, label);
                response.addDataset(choice(fields, Presenting.menu(context, face, label, account),
                        i < chips ? presenting.chip(i, face, label, account) : null, sender(context, fill, true)));
                offered = true;
            }
            if (search) {
                Intent pick = destination.write(new Intent(context, SearchActivity.class));
                CharSequence action = context.getString(R.string.autofill_search_title);
                CharSequence description = context.getString(R.string.autofill_search);
                // Gboard on Android 17 showed nothing for a response whose one chip was pinned.
                InlinePresentation chip = count > 0
                        ? presenting.pinned(Math.min(count, chips), description)
                        : presenting.entry(0, action, description);
                response.addDataset(choice(fields, Presenting.entryMenu(context, action), chip,
                        sender(context, pick, true)));
                offered = true;
            }
        }
        Bundle state = new Bundle();
        state.putString(REQUESTER, destination.requester.toString());
        JSONObject save = form.optJSONObject("save");
        AutofillId password = save != null ? screen.id(save.optInt("password", -1)) : null;
        if (password != null) {
            AutofillId username = screen.id(save.optInt("username", -1));
            response.setSaveInfo(saveInfo(username, password, form.optBoolean("email")));
            state.putParcelable(SAVE_USERNAME, username);
            state.putParcelable(SAVE_PASSWORD, password);
            offered = true;
        }
        if (!offered) {
            return null;
        }
        return response.setClientState(state).build();
    }

    /** null for none. */
    private static byte[] siteIcon(JSONObject icons, String site) {
        String icon = icons != null ? icons.optString(site) : "";
        if (icon.isEmpty()) {
            return null;
        }
        try {
            return Base64.getDecoder().decode(icon);
        } catch (IllegalArgumentException e) {
            return null;
        }
    }

    private static AutofillId[] fields(Screen screen, JSONObject form) {
        return Stream.concat(
                        Stream.of("username", "password").map(name -> screen.id(form.optInt(name, -1))),
                        screen.ids(form.optJSONArray("code")).stream())
                .filter(Objects::nonNull)
                .toArray(AutofillId[]::new);
    }

    /** Offers to save the username, or email, and the password once the form leaves the screen. */
    private static SaveInfo saveInfo(AutofillId username, AutofillId password, boolean email) {
        int types = SaveInfo.SAVE_DATA_TYPE_PASSWORD;
        if (username != null) {
            types |= email ? SaveInfo.SAVE_DATA_TYPE_EMAIL_ADDRESS : SaveInfo.SAVE_DATA_TYPE_USERNAME;
        }
        SaveInfo.Builder info = new SaveInfo.Builder(types, new AutofillId[] {password})
                .setFlags(SaveInfo.FLAG_SAVE_ON_ALL_VIEWS_INVISIBLE);
        if (username != null) {
            info.setOptionalIds(new AutofillId[] {username});
        }
        return info.build();
    }

    /** An entry that holds no value: choosing it starts authentication, whose answer fills the fields. */
    private static Dataset choice(AutofillId[] fields, RemoteViews menu, InlinePresentation chip,
            IntentSender authentication) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            Dataset.Builder dataset = new Dataset.Builder(presentations(menu, chip));
            for (AutofillId field : fields) {
                dataset.setField(field, null);
            }
            return dataset.setAuthentication(authentication).build();
        }
        return legacyChoice(fields, menu, chip, authentication);
    }

    @SuppressWarnings("deprecation")
    private static Dataset legacyChoice(AutofillId[] fields, RemoteViews menu, InlinePresentation chip,
            IntentSender authentication) {
        Dataset.Builder dataset = new Dataset.Builder(menu);
        for (AutofillId field : fields) {
            dataset.setValue(field, null);
        }
        if (chip != null) {
            dataset.setInlinePresentation(chip);
        }
        return dataset.setAuthentication(authentication).build();
    }

    @SuppressWarnings("deprecation")
    private static void authenticate(FillResponse.Builder response, AutofillId[] fields, IntentSender unlock,
            RemoteViews menu, InlinePresentation chip) {
        response.setAuthentication(fields, unlock, menu, chip);
    }

    @RequiresApi(Build.VERSION_CODES.TIRAMISU)
    private static Presentations presentations(RemoteViews menu, InlinePresentation chip) {
        Presentations.Builder presentations = new Presentations.Builder().setMenuPresentation(menu);
        if (chip != null) {
            presentations.setInlinePresentation(chip);
        }
        return presentations.build();
    }

    static IntentSender sender(Context context, Intent intent, boolean mutable) {
        return pending(context, intent, mutable).getIntentSender();
    }

    /** Each intent's own identity keeps a later PendingIntent from rewriting an earlier one's extras. */
    static PendingIntent pending(Context context, Intent intent, boolean mutable) {
        intent.setIdentifier(UUID.randomUUID().toString());
        int flags = mutable ? PendingIntent.FLAG_MUTABLE : PendingIntent.FLAG_IMMUTABLE;
        return PendingIntent.getActivity(context, 0, intent, flags);
    }

    /** A dataset that fills its values at once: what the fill and search screens answer with. */
    static final class Filled {
        private final Dataset.Builder dataset;
        private boolean empty = true;

        Filled(Context context, CharSequence label) {
            RemoteViews menu = Presenting.menu(context, label, null);
            dataset = Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU
                    ? new Dataset.Builder(presentations(menu, null))
                    : legacyBuilder(menu);
        }

        /** Fills value into field, unless either is missing. */
        void set(AutofillId field, String value) {
            if (field == null || value == null || value.isEmpty()) {
                return;
            }
            AutofillValue text = AutofillValue.forText(value);
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                dataset.setField(field, new Field.Builder().setValue(text).build());
            } else {
                legacySet(dataset, field, text);
            }
            empty = false;
        }

        /** null when it fills nothing. */
        Dataset build() {
            return empty ? null : dataset.build();
        }

        @SuppressWarnings("deprecation")
        private static Dataset.Builder legacyBuilder(RemoteViews menu) {
            return new Dataset.Builder(menu);
        }

        @SuppressWarnings("deprecation")
        private static void legacySet(Dataset.Builder dataset, AutofillId field, AutofillValue value) {
            dataset.setValue(field, value);
        }
    }
}
