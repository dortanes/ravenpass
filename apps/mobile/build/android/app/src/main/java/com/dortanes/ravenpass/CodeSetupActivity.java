package com.dortanes.ravenpass;

import android.app.Activity;
import android.content.Intent;
import android.net.Uri;
import android.os.Bundle;

import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.concurrent.Executor;
import java.util.concurrent.Executors;

/** Hands an opened otpauth link to the core and brings Ravenpass forward, where the owner adds it; it shows nothing. */
public final class CodeSetupActivity extends Activity {
    private static final Executor RECEIVER = Executors.newSingleThreadExecutor();

    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        Uri link = getIntent().getData();
        if (state == null && link != null) {
            byte[] linkUtf8 = link.toString().getBytes(StandardCharsets.UTF_8);
            RECEIVER.execute(() -> {
                try {
                    Bridge.codeSetup(linkUtf8);
                } finally {
                    Arrays.fill(linkUtf8, (byte) 0);
                }
            });
        }
        Intent launch = getPackageManager().getLaunchIntentForPackage(getPackageName());
        if (launch != null) {
            startActivity(launch);
        }
        finish();
    }
}
