package com.dortanes.ravenpass;

import android.app.Activity;
import android.app.Application;
import android.content.ActivityNotFoundException;
import android.content.ClipData;
import android.content.ClipDescription;
import android.content.ClipboardManager;
import android.content.ComponentName;
import android.content.ContentProviderClient;
import android.content.ContentResolver;
import android.content.Intent;
import android.content.pm.ApplicationInfo;
import android.content.pm.PackageManager;
import android.content.pm.ProviderInfo;
import android.content.res.Resources;
import android.credentials.CredentialManager;
import android.database.Cursor;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.CancellationSignal;
import android.os.Environment;
import android.os.Handler;
import android.os.Looper;
import android.os.ParcelFileDescriptor;
import android.os.PersistableBundle;
import android.os.storage.StorageManager;
import android.os.storage.StorageVolume;
import android.print.PageRange;
import android.print.PrintAttributes;
import android.print.PrintDocumentAdapter;
import android.print.PrintManager;
import android.provider.DocumentsContract;
import android.provider.OpenableColumns;
import android.provider.Settings;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyPermanentlyInvalidatedException;
import android.security.keystore.KeyProperties;
import android.view.autofill.AutofillManager;
import android.webkit.RenderProcessGoneDetail;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;

import androidx.activity.result.ActivityResult;
import androidx.activity.result.ActivityResultLauncher;
import androidx.activity.result.PickVisualMediaRequest;
import androidx.activity.result.contract.ActivityResultContract;
import androidx.activity.result.contract.ActivityResultContracts;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.biometric.BiometricManager;
import androidx.biometric.BiometricPrompt;
import androidx.core.content.ContextCompat;
import androidx.fragment.app.FragmentActivity;

import java.io.ByteArrayOutputStream;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.SyncFailedException;
import java.lang.ref.WeakReference;
import java.nio.charset.StandardCharsets;
import java.security.GeneralSecurityException;
import java.security.Key;
import java.security.KeyFactory;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.KeyStore;
import java.security.PrivateKey;
import java.security.ProviderException;
import java.security.PublicKey;
import java.security.UnrecoverableKeyException;
import java.security.spec.ECGenParameterSpec;
import java.security.spec.MGF1ParameterSpec;
import java.security.spec.X509EncodedKeySpec;
import java.util.Arrays;
import java.util.Collections;
import java.util.Set;
import java.util.UUID;
import java.util.concurrent.Callable;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.FutureTask;
import java.util.concurrent.atomic.AtomicReference;

import javax.crypto.Cipher;
import javax.crypto.KeyAgreement;
import javax.crypto.spec.OAEPParameterSpec;
import javax.crypto.spec.PSource;

import com.dortanes.ravenpass.autofill.CredentialService;
import com.wails.app.MainActivity;

/** Android services the Go core calls over JNI from its own threads; text crosses as UTF-8 bytes. */
final class Bridge {
    // Statuses shared with bridge.go.
    private static final int STATUS_OK = 0;
    private static final int STATUS_CANCELED = 1;
    private static final int STATUS_FAILED = 2;
    private static final int STATUS_UNAVAILABLE = 3;
    private static final int STATUS_REJECTED = 4;
    private static final int STATUS_ERROR = 5;
    private static final int STATUS_TOO_LARGE = 6;
    private static final int STATUS_GONE = 7;

    private static final String EXTERNAL_STORAGE = "com.android.externalstorage.documents";
    /** Document providers that keep files on the device, with no network behind a not-found. */
    private static final Set<String> LOCAL_PROVIDERS = Set.of(EXTERNAL_STORAGE,
            "com.android.providers.downloads.documents", "com.android.providers.media.documents");

    private static final String KEYSTORE = "AndroidKeyStore";
    private static final int PRESENCE_KEY_BITS = 2048;
    private static final String PRESENCE_CIPHER = "RSA/ECB/OAEPWithSHA-256AndMGF1Padding";
    // Android Keystore takes MGF1 with SHA-1 alone before API 35; keystore.go seals with the same parameters.
    private static final OAEPParameterSpec PRESENCE_OAEP = new OAEPParameterSpec("SHA-256", "MGF1",
            MGF1ParameterSpec.SHA1, PSource.PSpecified.DEFAULT);
    private static final int AUTHENTICATORS = BiometricManager.Authenticators.BIOMETRIC_STRONG
            | BiometricManager.Authenticators.DEVICE_CREDENTIAL;
    // ClipDescription.EXTRA_IS_SENSITIVE from API 33; earlier releases ignore the extra.
    private static final String EXTRA_IS_SENSITIVE = "android.content.extra.IS_SENSITIVE";
    private static final int DOCUMENT_ACCESS = Intent.FLAG_GRANT_READ_URI_PERMISSION
            | Intent.FLAG_GRANT_WRITE_URI_PERMISSION;
    private static final String VAULT_TYPE = "application/octet-stream";
    private static final int READ_BUFFER = 64 * 1024;
    // apps/mobile/Makefile writes the notices into src/main/assets.
    private static final String NOTICES_ASSET = "THIRD_PARTY_NOTICES.txt";

    private static final Handler MAIN = new Handler(Looper.getMainLooper());
    // One request of each kind at a time.
    private static final AtomicReference<Request> PROMPT = new AtomicReference<>();
    private static final AtomicReference<Request> PICK = new AtomicReference<>();
    private static final AtomicReference<Request> SCREEN = new AtomicReference<>();
    private static final AtomicReference<Request> PRINT = new AtomicReference<>();

    private static volatile Application application;
    // Main thread only.
    private static WeakReference<FragmentActivity> resumed = new WeakReference<>(null);
    private static volatile long lastWrite = -1;

    private Bridge() {
    }

    static void attach(Application app) {
        application = app;
        app.registerActivityLifecycleCallbacks(new Lifecycle());
        // Android 10 and later report clipboard changes only while the app is in front.
        clipboard().addPrimaryClipChangedListener(() -> dropReplacedScan(clipboard().getPrimaryClipDescription()));
        attach(app.getFilesDir().getAbsolutePath().getBytes(StandardCharsets.UTF_8));
    }

    private static native void attach(byte[] filesDirectoryUtf8);

    private static native void interfaceDestroyed();

    /** Holds an otpauth link until the owner adds it to a credential; it may build the core, so never on the main thread. */
    static native void codeSetup(byte[] linkUtf8);

    /**
     * Returns the X.509 SubjectPublicKeyInfo of a new key under the alias: a P-256 key agreement key, or for
     * presence an RSA decryption key that decrypt uses only within an owner verification made for that decryption.
     */
    static byte[] createKey(byte[] aliasUtf8, boolean presence) {
        if (presence && !ownerAvailable()) {
            return result(STATUS_UNAVAILABLE);
        }
        String alias = utf8(aliasUtf8);
        try {
            boolean strongBox = application.getPackageManager()
                    .hasSystemFeature(PackageManager.FEATURE_STRONGBOX_KEYSTORE);
            KeyPair pair;
            try {
                pair = generate(alias, presence, strongBox);
            } catch (ProviderException e) {
                // A StrongBox without the key's algorithm or purpose refuses it with a ProviderException.
                if (!strongBox) throw e;
                pair = generate(alias, presence, false);
            }
            return result(STATUS_OK, pair.getPublic().getEncoded());
        } catch (Exception e) {
            return result(STATUS_ERROR);
        }
    }

    /** Returns the raw ECDH secret of the key agreement key under the alias with the peer's SubjectPublicKeyInfo. */
    static byte[] agree(byte[] aliasUtf8, byte[] peerSpki) {
        try {
            PrivateKey key = privateKey(aliasUtf8, KeyProperties.KEY_ALGORITHM_EC);
            if (key == null) {
                return result(STATUS_REJECTED);
            }
            PublicKey peer = KeyFactory.getInstance(KeyProperties.KEY_ALGORITHM_EC)
                    .generatePublic(new X509EncodedKeySpec(peerSpki));
            KeyAgreement agreement = KeyAgreement.getInstance("ECDH", KEYSTORE);
            agreement.init(key);
            agreement.doPhase(peer, true);
            return secretResult(agreement.generateSecret());
        } catch (KeyPermanentlyInvalidatedException | UnrecoverableKeyException e) {
            return result(STATUS_REJECTED);
        } catch (Exception e) {
            return result(STATUS_ERROR);
        }
    }

    /** Returns the RSA-OAEP plaintext once the owner verifies, for the reason, this decryption alone. */
    static byte[] decrypt(byte[] aliasUtf8, byte[] ciphertext, byte[] reasonUtf8) {
        Cipher cipher;
        try {
            PrivateKey key = privateKey(aliasUtf8, KeyProperties.KEY_ALGORITHM_RSA);
            if (key == null) {
                return result(STATUS_REJECTED);
            }
            cipher = Cipher.getInstance(PRESENCE_CIPHER);
            cipher.init(Cipher.DECRYPT_MODE, key, PRESENCE_OAEP);
        } catch (KeyPermanentlyInvalidatedException | UnrecoverableKeyException e) {
            return result(STATUS_REJECTED);
        } catch (Exception e) {
            return result(STATUS_ERROR);
        }
        int verified = ask(new Prompt(utf8(reasonUtf8), new BiometricPrompt.CryptoObject(cipher)), PROMPT);
        if (verified != STATUS_OK) {
            return result(verified);
        }
        try {
            return secretResult(cipher.doFinal(ciphertext));
        } catch (GeneralSecurityException e) {
            return result(STATUS_ERROR);
        }
    }

    static boolean ownerAvailable() {
        return BiometricManager.from(application).canAuthenticate(AUTHENTICATORS)
                == BiometricManager.BIOMETRIC_SUCCESS;
    }

    static int authenticate(byte[] reasonUtf8) {
        return verifyOwner(utf8(reasonUtf8));
    }

    /** The call waiting on the dismissed prompt returns STATUS_CANCELED. */
    static void cancelAuthentication() {
        Request attempt = PROMPT.get();
        if (attempt != null) {
            MAIN.post(attempt::cancel);
        }
    }

    /** Returns the clip's timestamp, or -1 with nothing of the text left on the clipboard. */
    static long writeText(byte[] textUtf8) throws Exception {
        String text = utf8(textUtf8);
        return onMainThread(() -> {
            ClipProvider.drop();
            return place(ClipData.newPlainText(label(), text));
        });
    }

    /** The clip holds a ClipProvider address serving a copy of the content; the caller wipes its array. */
    static long writeScan(byte[] content, byte[] mediaTypeUtf8) throws Exception {
        byte[] held = content.clone();
        String mediaType = utf8(mediaTypeUtf8);
        return onMainThread(() -> {
            Uri scan = ClipProvider.hold(application, held, mediaType, label());
            return place(new ClipData(label(), new String[] {mediaType}, new ClipData.Item(scan)));
        });
    }

    /** Android 10 and later hide the clipboard from a background app; the last write's timestamp stands in. */
    static long changeCount() {
        try {
            return onMainThread(() -> {
                ClipDescription current = clipboard().getPrimaryClipDescription();
                if (current == null) {
                    return lastWrite;
                }
                dropReplacedScan(current);
                return current.getTimestamp();
            });
        } catch (Exception e) {
            return lastWrite;
        }
    }

    static void clearClipboard() throws Exception {
        onMainThread(() -> {
            ClipProvider.drop();
            clipboard().clearPrimaryClip();
            return null;
        });
    }

    static void allowScreenshots(boolean allowed) {
        MAIN.post(() -> ScreenCapture.allowInMainWindow(allowed));
    }

    /** Returns STATUS_OK once the owner leaves the print dialog; the caller wipes the page's array. */
    static int print(byte[] jobUtf8, byte[] pageUtf8) {
        return ask(new PrintPage(utf8(jobUtf8), utf8(pageUtf8)), PRINT);
    }

    /** Returns the kept document's address, name and provider label, separated by NUL. */
    static byte[] pickDocument(boolean create, byte[] nameUtf8) {
        Pick pick = Pick.document(create, utf8(nameUtf8), create ? VAULT_TYPE : "*/*");
        int picked = ask(pick, PICK);
        if (picked != STATUS_OK) {
            return result(picked);
        }
        try {
            return keep(pick.chosen);
        } catch (Exception e) {
            return result(STATUS_ERROR);
        }
    }

    /** Returns the created document's address; access to it lasts until the device restarts. */
    static byte[] createDocument(byte[] nameUtf8, byte[] typeUtf8) {
        Pick pick = Pick.document(true, utf8(nameUtf8), utf8(typeUtf8));
        int picked = ask(pick, PICK);
        if (picked != STATUS_OK) {
            return result(picked);
        }
        return result(STATUS_OK, pick.chosen.toString().getBytes(StandardCharsets.UTF_8));
    }

    /** Returns the kept folder's address, name and provider label, separated by NUL. */
    static byte[] pickFolder() {
        Pick pick = Pick.folder();
        int picked = ask(pick, PICK);
        if (picked != STATUS_OK) {
            return result(picked);
        }
        try {
            return keepFolder(pick.chosen);
        } catch (Exception e) {
            return result(STATUS_ERROR);
        }
    }

    /** The provider may rename the document, such as with a number where the name is taken. */
    static byte[] createInFolder(byte[] folderUtf8, byte[] nameUtf8, byte[] typeUtf8) {
        try {
            Uri created = DocumentsContract.createDocument(resolver(), folderRoot(document(folderUtf8)),
                    utf8(typeUtf8), utf8(nameUtf8));
            if (created == null) {
                return result(STATUS_ERROR);
            }
            return result(STATUS_OK, created.toString().getBytes(StandardCharsets.UTF_8));
        } catch (SecurityException e) {
            // The owner revoked the folder's access, or its provider no longer grants it.
            return result(STATUS_REJECTED);
        } catch (Exception e) {
            return result(STATUS_ERROR);
        }
    }

    /** Returns STATUS_GONE where opening the document fails as not found and gone confirms it. */
    static byte[] readDocument(byte[] addressUtf8, long limit) {
        try {
            ParcelFileDescriptor descriptor = resolver().openFileDescriptor(document(addressUtf8), "r");
            if (descriptor == null) {
                return result(STATUS_ERROR);
            }
            try (InputStream input = new ParcelFileDescriptor.AutoCloseInputStream(descriptor)) {
                if (descriptor.getStatSize() > limit) {
                    return result(STATUS_TOO_LARGE);
                }
                // A pipe reports no size, so the read counts too.
                ByteArrayOutputStream content = new ByteArrayOutputStream();
                byte[] buffer = new byte[READ_BUFFER];
                for (int read = input.read(buffer); read != -1; read = input.read(buffer)) {
                    if (content.size() + (long) read > limit) {
                        return result(STATUS_TOO_LARGE);
                    }
                    content.write(buffer, 0, read);
                }
                return result(STATUS_OK, content.toByteArray());
            }
        } catch (FileNotFoundException e) {
            return result(gone(document(addressUtf8)) ? STATUS_GONE : STATUS_ERROR);
        } catch (Exception e) {
            return result(STATUS_ERROR);
        }
    }

    /** A cloud provider uploads the document on close, so a failed close fails the write. */
    static int writeDocument(byte[] addressUtf8, byte[] content) {
        try {
            ParcelFileDescriptor descriptor = resolver().openFileDescriptor(document(addressUtf8), "wt");
            if (descriptor == null) {
                return STATUS_ERROR;
            }
            try (FileOutputStream output = new ParcelFileDescriptor.AutoCloseOutputStream(descriptor)) {
                output.write(content);
                output.flush();
                try {
                    output.getFD().sync();
                } catch (SyncFailedException e) {
                    // A provider may hand out a pipe, which cannot be synced.
                }
            }
            return STATUS_OK;
        } catch (Exception e) {
            return STATUS_ERROR;
        }
    }

    /** Returns STATUS_GONE where the delete fails and the folder the document was created in no longer lists it. */
    static int deleteDocument(byte[] addressUtf8) {
        Uri document = document(addressUtf8);
        boolean deleted;
        try {
            deleted = DocumentsContract.deleteDocument(resolver(), document);
        } catch (Exception e) {
            deleted = false;
        }
        if (!deleted && !absentFromFolder(document)) {
            return STATUS_ERROR;
        }
        release(document);
        return deleted ? STATUS_OK : STATUS_GONE;
    }

    /** Returns the picture's address, readable until the app stops, and its name, separated by NUL. */
    static byte[] pickPhoto() {
        PhotoPick pick = new PhotoPick();
        int picked = ask(pick, PICK);
        if (picked != STATUS_OK) {
            return result(picked);
        }
        try {
            String photo = pick.chosen + "\0" + photoName(pick.chosen);
            return result(STATUS_OK, photo.getBytes(StandardCharsets.UTF_8));
        } catch (RuntimeException e) {
            return result(STATUS_ERROR);
        }
    }

    static boolean autofillSelected() {
        AutofillManager autofill = application.getSystemService(AutofillManager.class);
        return autofill != null && autofill.hasEnabledAutofillServices();
    }

    /** Returns 1 enabled, 0 not, and -1 where the device cannot say, as before Android 14. */
    static int passkeyProvider() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
            return -1;
        }
        CredentialManager credentials = application.getSystemService(CredentialManager.class);
        if (credentials == null) {
            return -1;
        }
        try {
            return credentials.isEnabledCredentialProviderService(
                    new ComponentName(application, CredentialService.class)) ? 1 : 0;
        } catch (RuntimeException e) {
            return -1;
        }
    }

    /** Returns STATUS_OK once the owner leaves the screen. */
    static int selectAutofill() {
        return ask(new SelectService(), SCREEN);
    }

    /** Returns the device's BCP 47 tags separated by commas, most preferred first, whatever language the app uses. */
    static byte[] preferredLanguages() {
        return Resources.getSystem().getConfiguration().getLocales().toLanguageTags()
                .getBytes(StandardCharsets.UTF_8);
    }

    /** An empty tag follows the device's language. */
    static void setLanguage(byte[] tagUtf8) {
        AppLanguage.follow(application, utf8(tagUtf8));
    }

    /** "light", "dark" or "system". */
    static void setAppearance(byte[] appearanceUtf8) {
        AppAppearance.follow(application, utf8(appearanceUtf8));
    }

    /** Returns the name an installed app shows, empty for a package this device does not have or cannot see. */
    static byte[] appName(byte[] packageUtf8) {
        PackageManager packages = application.getPackageManager();
        try {
            return packages.getApplicationLabel(applicationInfo(packages, utf8(packageUtf8))).toString()
                    .getBytes(StandardCharsets.UTF_8);
        } catch (PackageManager.NameNotFoundException e) {
            return new byte[0];
        }
    }

    @SuppressWarnings("deprecation")
    private static ApplicationInfo applicationInfo(PackageManager packages, String packageName)
            throws PackageManager.NameNotFoundException {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            return packages.getApplicationInfo(packageName, PackageManager.ApplicationInfoFlags.of(0));
        }
        return packages.getApplicationInfo(packageName, 0);
    }

    /** Returns the Android release and API level, such as "15 (API 35)". */
    static byte[] systemVersion() {
        return (Build.VERSION.RELEASE + " (API " + Build.VERSION.SDK_INT + ")").getBytes(StandardCharsets.UTF_8);
    }

    /** Returns STATUS_GONE for an APK built without the notices asset. */
    static byte[] thirdPartyNotices() {
        try (InputStream input = application.getAssets().open(NOTICES_ASSET)) {
            ByteArrayOutputStream content = new ByteArrayOutputStream();
            byte[] buffer = new byte[READ_BUFFER];
            for (int read = input.read(buffer); read != -1; read = input.read(buffer)) {
                content.write(buffer, 0, read);
            }
            return result(STATUS_OK, content.toByteArray());
        } catch (FileNotFoundException e) {
            return result(STATUS_GONE);
        } catch (IOException e) {
            return result(STATUS_ERROR);
        }
    }

    private static KeyPair generate(String alias, boolean presence, boolean strongBox)
            throws GeneralSecurityException {
        String algorithm;
        KeyGenParameterSpec.Builder spec;
        if (presence) {
            algorithm = KeyProperties.KEY_ALGORITHM_RSA;
            // A zero timeout authorizes only the operation whose CryptoObject the prompt carried.
            spec = new KeyGenParameterSpec.Builder(alias, KeyProperties.PURPOSE_DECRYPT)
                    .setKeySize(PRESENCE_KEY_BITS)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_RSA_OAEP)
                    .setDigests(KeyProperties.DIGEST_SHA256)
                    .setUserAuthenticationRequired(true)
                    .setUserAuthenticationParameters(0,
                            KeyProperties.AUTH_BIOMETRIC_STRONG | KeyProperties.AUTH_DEVICE_CREDENTIAL)
                    .setInvalidatedByBiometricEnrollment(false);
        } else {
            algorithm = KeyProperties.KEY_ALGORITHM_EC;
            spec = new KeyGenParameterSpec.Builder(alias, KeyProperties.PURPOSE_AGREE_KEY)
                    .setAlgorithmParameterSpec(new ECGenParameterSpec("secp256r1"));
        }
        spec.setUnlockedDeviceRequired(true).setIsStrongBoxBacked(strongBox);
        KeyPairGenerator generator = KeyPairGenerator.getInstance(algorithm, KEYSTORE);
        generator.initialize(spec.build());
        return generator.generateKeyPair();
    }

    // Returns null where the alias holds no private key of the algorithm.
    private static PrivateKey privateKey(byte[] aliasUtf8, String algorithm)
            throws GeneralSecurityException, IOException {
        KeyStore keyStore = KeyStore.getInstance(KEYSTORE);
        keyStore.load(null);
        Key key = keyStore.getKey(utf8(aliasUtf8), null);
        return key instanceof PrivateKey && algorithm.equals(key.getAlgorithm()) ? (PrivateKey) key : null;
    }

    private static byte[] secretResult(byte[] secret) {
        try {
            return result(STATUS_OK, secret);
        } finally {
            Arrays.fill(secret, (byte) 0);
        }
    }

    private static int verifyOwner(String reason) {
        return ask(new Prompt(reason, null), PROMPT);
    }

    // The caller must not be the main thread, which shows the request the caller waits on.
    private static int ask(Request request, AtomicReference<Request> slot) {
        if (Looper.myLooper() == Looper.getMainLooper() || !slot.compareAndSet(null, request)) {
            return STATUS_ERROR;
        }
        try {
            MAIN.post(request::show);
            return request.await();
        } finally {
            slot.set(null);
        }
    }

    private static byte[] keep(Uri document) {
        String name;
        try (Cursor cursor = resolver().query(document, new String[] {
                DocumentsContract.Document.COLUMN_DISPLAY_NAME, DocumentsContract.Document.COLUMN_FLAGS},
                null, null, null)) {
            if (cursor == null || !cursor.moveToFirst() || cursor.isNull(0)) {
                return result(STATUS_ERROR);
            }
            if ((cursor.getInt(1) & DocumentsContract.Document.FLAG_SUPPORTS_WRITE) == 0) {
                return result(STATUS_REJECTED);
            }
            name = cursor.getString(0);
        }
        try {
            resolver().takePersistableUriPermission(document, DOCUMENT_ACCESS);
        } catch (SecurityException e) {
            return result(STATUS_REJECTED);
        }
        String picked = String.join("\0", document.toString(), name, place(document.getAuthority()));
        return result(STATUS_OK, picked.getBytes(StandardCharsets.UTF_8));
    }

    private static byte[] keepFolder(Uri tree) {
        try {
            resolver().takePersistableUriPermission(tree, DOCUMENT_ACCESS);
        } catch (SecurityException e) {
            return result(STATUS_REJECTED);
        }
        boolean kept = false;
        try {
            String name;
            try (Cursor cursor = resolver().query(folderRoot(tree), new String[] {
                    DocumentsContract.Document.COLUMN_DISPLAY_NAME, DocumentsContract.Document.COLUMN_FLAGS},
                    null, null, null)) {
                if (cursor == null || !cursor.moveToFirst() || cursor.isNull(0)) {
                    return result(STATUS_ERROR);
                }
                if ((cursor.getInt(1) & DocumentsContract.Document.FLAG_DIR_SUPPORTS_CREATE) == 0) {
                    return result(STATUS_REJECTED);
                }
                name = cursor.getString(0);
            }
            String picked = String.join("\0", tree.toString(), name, place(tree.getAuthority()));
            kept = true;
            return result(STATUS_OK, picked.getBytes(StandardCharsets.UTF_8));
        } finally {
            if (!kept) {
                release(tree);
            }
        }
    }

    // The folder a tree address grants, as a document.
    private static Uri folderRoot(Uri tree) {
        return DocumentsContract.buildDocumentUriUsingTree(tree, DocumentsContract.getTreeDocumentId(tree));
    }

    private static void release(Uri granted) {
        try {
            resolver().releasePersistableUriPermission(granted, DOCUMENT_ACCESS);
        } catch (SecurityException e) {
            // A deleted document loses its grants, and one created in a folder has none of its own.
        }
    }

    // The provider's label, such as a cloud drive's name, else its authority.
    private static String place(String authority) {
        PackageManager packages = application.getPackageManager();
        ProviderInfo provider = packages.resolveContentProvider(authority, 0);
        return provider != null ? provider.loadLabel(packages).toString() : authority;
    }

    private static String photoName(Uri photo) {
        try (Cursor cursor = resolver().query(photo, new String[] {OpenableColumns.DISPLAY_NAME}, null, null, null)) {
            if (cursor != null && cursor.moveToFirst() && !cursor.isNull(0) && !cursor.getString(0).isEmpty()) {
                return cursor.getString(0);
            }
        }
        String segment = photo.getLastPathSegment();
        return segment != null ? segment : "";
    }

    // A document created in a picked folder is gone for certain where the folder's complete listing lacks it; a listing
    // the provider is still loading, such as a cloud drive's, proves nothing.
    private static boolean absentFromFolder(Uri document) {
        if (!DocumentsContract.isTreeUri(document)) {
            return false;
        }
        try {
            String id = DocumentsContract.getDocumentId(document);
            Uri children = DocumentsContract.buildChildDocumentsUriUsingTree(document,
                    DocumentsContract.getTreeDocumentId(document));
            try (Cursor cursor = resolver().query(children,
                    new String[] {DocumentsContract.Document.COLUMN_DOCUMENT_ID}, null, null, null)) {
                if (cursor == null || cursor.getExtras().getBoolean(DocumentsContract.EXTRA_LOADING)) {
                    return false;
                }
                while (cursor.moveToNext()) {
                    if (id.equals(cursor.getString(0))) {
                        return false;
                    }
                }
                return true;
            }
        } catch (Exception e) {
            return false;
        }
    }

    // A cloud provider's not-found can mean offline: there only a query answered with zero rows is definite, since
    // DocumentsProvider.query answers every queryDocument that throws FileNotFoundException with no cursor. A local
    // provider has no network behind it, so the caller's not-found is definite while the document's volume is mounted.
    private static boolean gone(Uri document) {
        if (!DocumentsContract.isDocumentUri(application, document)) {
            return false;
        }
        if (LOCAL_PROVIDERS.contains(document.getAuthority())) {
            return mounted(document);
        }
        try (ContentProviderClient provider = resolver().acquireUnstableContentProviderClient(document)) {
            if (provider == null) {
                return false;
            }
            try (Cursor cursor = provider.query(document,
                    new String[] {DocumentsContract.Document.COLUMN_DOCUMENT_ID}, null, null, null)) {
                return cursor != null && !cursor.moveToFirst();
            }
        } catch (Exception e) {
            return false;
        }
    }

    // The external storage provider names a document "volume:path", the volume being "primary", "home" for the primary
    // volume's Documents folder, or a removable one's UUID; the downloads and media providers keep theirs on the primary
    // volume.
    private static boolean mounted(Uri document) {
        if (!EXTERNAL_STORAGE.equals(document.getAuthority())) {
            return true;
        }
        String id = DocumentsContract.getDocumentId(document);
        int colon = id.indexOf(':');
        if (colon < 0) {
            return false;
        }
        String volume = id.substring(0, colon);
        StorageManager storage = application.getSystemService(StorageManager.class);
        for (StorageVolume candidate : storage.getStorageVolumes()) {
            boolean named = candidate.isPrimary()
                    ? "primary".equals(volume) || "home".equals(volume)
                    : volume.equals(candidate.getUuid());
            if (named) {
                return Environment.MEDIA_MOUNTED.equals(candidate.getState());
            }
        }
        return false;
    }

    private static int errorStatus(int code) {
        return switch (code) {
            case BiometricPrompt.ERROR_CANCELED, BiometricPrompt.ERROR_USER_CANCELED,
                    BiometricPrompt.ERROR_NEGATIVE_BUTTON -> STATUS_CANCELED;
            case BiometricPrompt.ERROR_LOCKOUT, BiometricPrompt.ERROR_LOCKOUT_PERMANENT,
                    BiometricPrompt.ERROR_TIMEOUT -> STATUS_FAILED;
            case BiometricPrompt.ERROR_NO_BIOMETRICS, BiometricPrompt.ERROR_NO_DEVICE_CREDENTIAL,
                    BiometricPrompt.ERROR_HW_UNAVAILABLE, BiometricPrompt.ERROR_HW_NOT_PRESENT,
                    BiometricPrompt.ERROR_SECURITY_UPDATE_REQUIRED -> STATUS_UNAVAILABLE;
            default -> STATUS_ERROR;
        };
    }

    private static <T> T onMainThread(Callable<T> task) throws Exception {
        if (Looper.myLooper() == Looper.getMainLooper()) {
            return task.call();
        }
        FutureTask<T> future = new FutureTask<>(task);
        MAIN.post(future);
        return future.get();
    }

    private static ClipboardManager clipboard() {
        return application.getSystemService(ClipboardManager.class);
    }

    // Main thread only. Returns -1 with the clipboard emptied where the clipboard did not take the clip.
    private static long place(ClipData clip) {
        ClipboardManager clipboard = clipboard();
        PersistableBundle extras = new PersistableBundle();
        extras.putBoolean(EXTRA_IS_SENSITIVE, true);
        clip.getDescription().setExtras(extras);
        clipboard.setPrimaryClip(clip);
        ClipDescription placed = clipboard.getPrimaryClipDescription();
        if (placed == null) {
            ClipProvider.drop();
            clipboard.clearPrimaryClip();
            return -1L;
        }
        lastWrite = placed.getTimestamp();
        return lastWrite;
    }

    // Main thread only. A clip placed since the app's last copy, by any app, ends the served scan.
    private static void dropReplacedScan(ClipDescription current) {
        if (current == null || current.getTimestamp() != lastWrite) {
            ClipProvider.drop();
        }
    }

    private static ContentResolver resolver() {
        return application.getContentResolver();
    }

    private static Uri document(byte[] addressUtf8) {
        return Uri.parse(utf8(addressUtf8));
    }

    private static CharSequence label() {
        return application.getApplicationInfo().loadLabel(application.getPackageManager());
    }

    private static String utf8(byte[] bytes) {
        return new String(bytes, StandardCharsets.UTF_8);
    }

    private static byte[] result(int status) {
        return new byte[] {(byte) status};
    }

    private static byte[] result(int status, byte[] payload) {
        byte[] result = new byte[1 + payload.length];
        result[0] = (byte) status;
        System.arraycopy(payload, 0, result, 1, payload.length);
        return result;
    }

    /** Shown and answered on the main thread, awaited on the caller's; its activity's end ends it. */
    private abstract static class Request {
        private final CountDownLatch answered = new CountDownLatch(1);
        private int status = STATUS_ERROR;
        // Main thread only.
        private Activity host;
        private boolean finished;

        /** The answer arrives through finish. */
        abstract void present(FragmentActivity activity);

        abstract void release();

        final void show() {
            if (finished) {
                return;
            }
            FragmentActivity activity = resumed.get();
            if (activity == null) {
                finish(STATUS_UNAVAILABLE);
                return;
            }
            host = activity;
            try {
                present(activity);
            } catch (RuntimeException e) {
                finish(STATUS_ERROR);
            }
        }

        void cancel() {
            finish(STATUS_CANCELED);
        }

        final void hostDestroyed(Activity activity) {
            if (activity == host) {
                cancel();
            }
        }

        final int await() {
            try {
                answered.await();
                return status;
            } catch (InterruptedException e) {
                MAIN.post(this::cancel);
                Thread.currentThread().interrupt();
                return STATUS_CANCELED;
            }
        }

        final void finish(int result) {
            if (finished) {
                return;
            }
            finished = true;
            host = null;
            release();
            status = result;
            answered.countDown();
        }
    }

    /** The reason is a title, then optionally a line feed and a description. */
    private static final class Prompt extends Request {
        private final String reason;
        @Nullable private final BiometricPrompt.CryptoObject crypto;
        // Main thread only.
        private BiometricPrompt biometricPrompt;

        /** A prompt with crypto authorizes that operation alone. */
        Prompt(String reason, @Nullable BiometricPrompt.CryptoObject crypto) {
            this.reason = reason;
            this.crypto = crypto;
        }

        @Override
        void present(FragmentActivity activity) {
            // BiometricPrompt drops a request made after its activity saved its state, and never answers it.
            if (activity.getSupportFragmentManager().isStateSaved()) {
                finish(STATUS_UNAVAILABLE);
                return;
            }
            biometricPrompt = new BiometricPrompt(activity, ContextCompat.getMainExecutor(activity),
                    new BiometricPrompt.AuthenticationCallback() {
                        @Override
                        public void onAuthenticationSucceeded(@NonNull BiometricPrompt.AuthenticationResult result) {
                            finish(STATUS_OK);
                        }

                        @Override
                        public void onAuthenticationError(int code, @NonNull CharSequence message) {
                            finish(errorStatus(code));
                        }
                    });
            String[] parts = reason.split("\n", 2);
            BiometricPrompt.PromptInfo.Builder info = new BiometricPrompt.PromptInfo.Builder()
                    .setTitle(parts[0])
                    .setAllowedAuthenticators(AUTHENTICATORS);
            if (parts.length > 1) {
                info.setDescription(parts[1]);
            }
            if (crypto != null) {
                biometricPrompt.authenticate(info.build(), crypto);
            } else {
                biometricPrompt.authenticate(info.build());
            }
        }

        @Override
        void cancel() {
            if (biometricPrompt != null) {
                biometricPrompt.cancelAuthentication();
            }
            super.cancel();
        }

        @Override
        void release() {
            biometricPrompt = null;
        }
    }

    private abstract static class Launch<I, O> extends Request {
        private final ActivityResultContract<I, O> contract;
        // Main thread only.
        private ActivityResultLauncher<I> launcher;

        Launch(ActivityResultContract<I, O> contract) {
            this.contract = contract;
        }

        abstract I input();

        /** Ends the request with finish. */
        abstract void answered(O result);

        @Override
        final void present(FragmentActivity activity) {
            // A key per launch keeps a result that reaches a recreated activity from answering a later launch.
            launcher = activity.getActivityResultRegistry().register(
                    "com.dortanes.ravenpass.launch." + UUID.randomUUID(), contract, this::answered);
            try {
                launcher.launch(input());
            } catch (ActivityNotFoundException e) {
                finish(STATUS_UNAVAILABLE);
            }
        }

        @Override
        final void release() {
            if (launcher != null) {
                // Unregistering inside the result callback keeps the key's request code.
                MAIN.post(launcher::unregister);
                launcher = null;
            }
        }
    }

    private abstract static class IntentLaunch extends Launch<Intent, ActivityResult> {
        IntentLaunch() {
            super(new ActivityResultContracts.StartActivityForResult());
        }
    }

    private static final class Pick extends IntentLaunch {
        private final Intent intent;
        // Written on the main thread before the pick ends.
        private Uri chosen;

        private Pick(Intent intent) {
            this.intent = intent;
        }

        static Pick document(boolean create, String name, String type) {
            Intent intent = new Intent(create ? Intent.ACTION_CREATE_DOCUMENT : Intent.ACTION_OPEN_DOCUMENT)
                    .addCategory(Intent.CATEGORY_OPENABLE)
                    .setType(type);
            return new Pick(create ? intent.putExtra(Intent.EXTRA_TITLE, name) : intent);
        }

        /** The picked address is a tree that grants the documents inside the folder. */
        static Pick folder() {
            return new Pick(new Intent(Intent.ACTION_OPEN_DOCUMENT_TREE));
        }

        @Override
        Intent input() {
            return intent;
        }

        @Override
        void answered(ActivityResult result) {
            Intent data = result.getData();
            chosen = result.getResultCode() == Activity.RESULT_OK && data != null ? data.getData() : null;
            finish(chosen != null ? STATUS_OK : STATUS_CANCELED);
        }
    }

    private static final class PhotoPick extends Launch<PickVisualMediaRequest, Uri> {
        // Written on the main thread before the pick ends.
        private Uri chosen;

        PhotoPick() {
            super(new ActivityResultContracts.PickVisualMedia());
        }

        @Override
        PickVisualMediaRequest input() {
            return new PickVisualMediaRequest.Builder()
                    .setMediaType(ActivityResultContracts.PickVisualMedia.ImageOnly.INSTANCE)
                    .build();
        }

        @Override
        void answered(Uri result) {
            chosen = result;
            finish(chosen != null ? STATUS_OK : STATUS_CANCELED);
        }
    }

    // ACTION_CREDENTIAL_PROVIDER is public from API 35. Android 17 resolves it only with a package.
    private static final class SelectService extends IntentLaunch {
        @Override
        Intent input() {
            String action = Build.VERSION.SDK_INT >= Build.VERSION_CODES.VANILLA_ICE_CREAM
                    ? Settings.ACTION_CREDENTIAL_PROVIDER
                    : Settings.ACTION_REQUEST_SET_AUTOFILL_SERVICE;
            return new Intent(action).setData(Uri.parse("package:" + application.getPackageName()));
        }

        @Override
        void answered(ActivityResult result) {
            finish(STATUS_OK);
        }
    }

    /** Prints one page from an offscreen WebView that runs no script and loads nothing beyond the page. */
    private static final class PrintPage extends Request {
        private final String job;
        // Main thread only; the page is dropped once the WebView holds it.
        private String page;
        private WebView view;
        private boolean printing;

        PrintPage(String job, String page) {
            this.job = job;
            this.page = page;
        }

        @Override
        void present(FragmentActivity activity) {
            PrintManager printer = activity.getSystemService(PrintManager.class);
            if (printer == null) {
                finish(STATUS_UNAVAILABLE);
                return;
            }
            view = new WebView(activity);
            WebSettings settings = view.getSettings();
            WebViewPolicy.restrict(settings);
            settings.setJavaScriptEnabled(false);
            settings.setBlockNetworkLoads(true);
            settings.setCacheMode(WebSettings.LOAD_NO_CACHE);
            view.setWebViewClient(new WebViewClient() {
                @Override
                public boolean shouldOverrideUrlLoading(WebView loading, WebResourceRequest request) {
                    return true;
                }

                @Override
                public WebResourceResponse shouldInterceptRequest(WebView loading, WebResourceRequest request) {
                    return request.isForMainFrame() ? null : new WebResourceResponse("text/plain", "UTF-8", 403,
                            "Blocked", Collections.emptyMap(), null);
                }

                @Override
                public void onPageFinished(WebView loaded, String url) {
                    if (loaded != view || printing) {
                        return;
                    }
                    printing = true;
                    try {
                        printer.print(job, new Finishing(loaded.createPrintDocumentAdapter(job)), null);
                    } catch (RuntimeException e) {
                        printing = false;
                        finish(STATUS_ERROR);
                    }
                }

                @Override
                public boolean onRenderProcessGone(WebView gone, RenderProcessGoneDetail detail) {
                    discard();
                    if (!printing) {
                        finish(STATUS_ERROR);
                    }
                    return true;
                }
            });
            view.loadDataWithBaseURL(null, page, "text/html", "UTF-8", null);
            page = null;
        }

        @Override
        void release() {
            page = null;
            if (!printing) {
                discard();
            }
        }

        private void discard() {
            if (view != null) {
                view.stopLoading();
                view.destroy();
                view = null;
            }
        }

        /** Passes the WebView's pages through and ends the request once the dialog closes. */
        private final class Finishing extends PrintDocumentAdapter {
            private final PrintDocumentAdapter pages;

            Finishing(PrintDocumentAdapter pages) {
                this.pages = pages;
            }

            @Override
            public void onStart() {
                pages.onStart();
            }

            @Override
            public void onLayout(PrintAttributes oldAttributes, PrintAttributes newAttributes,
                    CancellationSignal cancellation, LayoutResultCallback callback, Bundle extras) {
                pages.onLayout(oldAttributes, newAttributes, cancellation, callback, extras);
            }

            @Override
            public void onWrite(PageRange[] ranges, ParcelFileDescriptor destination, CancellationSignal cancellation,
                    WriteResultCallback callback) {
                pages.onWrite(ranges, destination, cancellation, callback);
            }

            @Override
            public void onFinish() {
                pages.onFinish();
                discard();
                finish(STATUS_OK);
            }
        }
    }

    private static final class Lifecycle extends ActivityCallbacks {
        @Override
        public void onActivityResumed(@NonNull Activity activity) {
            if (activity instanceof FragmentActivity) {
                resumed = new WeakReference<>((FragmentActivity) activity);
            }
        }

        @Override
        public void onActivityPaused(@NonNull Activity activity) {
            if (resumed.get() == activity) {
                resumed = new WeakReference<>(null);
            }
        }

        @Override
        public void onActivityDestroyed(@NonNull Activity activity) {
            if (activity instanceof MainActivity) {
                interfaceDestroyed();
            }
            hostDestroyed(PROMPT, activity);
            hostDestroyed(PICK, activity);
            hostDestroyed(SCREEN, activity);
            hostDestroyed(PRINT, activity);
        }

        private static void hostDestroyed(AtomicReference<Request> slot, Activity activity) {
            Request request = slot.get();
            if (request != null) {
                request.hostDestroyed(activity);
            }
        }
    }
}
