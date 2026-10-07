//go:build android

package bridge

/*
#include "jni_android.h"
*/
import "C"

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

var (
	attached       atomic.Bool
	filesDirectory atomic.Pointer[string]
)

// Java_com_dortanes_ravenpass_Bridge_attach implements Bridge.attach(byte[]), the UTF-8 files directory.
//
//export Java_com_dortanes_ravenpass_Bridge_attach
func Java_com_dortanes_ravenpass_Bridge_attach(env *C.JNIEnv, bridge C.jclass, directory C.jbyteArray) {
	if attached.Load() {
		return
	}
	var copied payload
	copied.data = C.ravenpass_bridge_copy(env, directory, &copied.length)
	path := string(copied.take())
	filesDirectory.Store(&path)
	if C.ravenpass_bridge_attach(env, bridge) == 0 {
		attached.Store(true)
	}
}

// Java_com_dortanes_ravenpass_autofill_Core_call implements Core.call(byte[]) with the ServeAutofill server.
//
//export Java_com_dortanes_ravenpass_autofill_Core_call
func Java_com_dortanes_ravenpass_autofill_Core_call(env *C.JNIEnv, _ C.jclass, request C.jbyteArray) C.jbyteArray {
	// cgo types JNI object references as integers, whose zero is Java's null.
	var none C.jbyteArray
	serve := autofillServer.Load()
	if serve == nil {
		return none
	}
	var copied payload
	copied.data = C.ravenpass_bridge_copy(env, request, &copied.length)
	in := copied.take()
	if in == nil {
		return none
	}
	defer clear(in)
	out := (*serve)(in)
	if len(out) == 0 {
		return none
	}
	defer clear(out)
	return C.ravenpass_bridge_array(env, unsafe.Pointer(unsafe.SliceData(out)), C.jsize(len(out)))
}

// Java_com_dortanes_ravenpass_Bridge_interfaceDestroyed implements Bridge.interfaceDestroyed() with OnInterfaceDestroyed.
//
//export Java_com_dortanes_ravenpass_Bridge_interfaceDestroyed
func Java_com_dortanes_ravenpass_Bridge_interfaceDestroyed(env *C.JNIEnv, bridge C.jclass) {
	if destroyed := interfaceDestroyed.Load(); destroyed != nil {
		(*destroyed)()
	}
}

// Java_com_dortanes_ravenpass_Bridge_codeSetup implements Bridge.codeSetup(byte[]), a UTF-8 otpauth link.
//
//export Java_com_dortanes_ravenpass_Bridge_codeSetup
func Java_com_dortanes_ravenpass_Bridge_codeSetup(env *C.JNIEnv, _ C.jclass, link C.jbyteArray) {
	var copied payload
	copied.data = C.ravenpass_bridge_copy(env, link, &copied.length)
	received := copied.take()
	defer clear(received)
	codeSetupOpened(received)
}

// FilesDirectory is the app's private files directory, empty until the application attaches.
func FilesDirectory() string {
	if path := filesDirectory.Load(); path != nil {
		return *path
	}
	return ""
}

// thread is the JNI environment of a goroutine locked to its OS thread until leave.
type thread struct {
	env          *C.JNIEnv
	attachedHere C.bool
}

// enter fails with statusUnavailable before Bridge attaches and statusError where JNI cannot attach the thread.
func enter() (thread, status) {
	if !attached.Load() {
		return thread{}, statusUnavailable
	}
	runtime.LockOSThread()
	var attachedHere C.bool
	env := C.ravenpass_bridge_enter(&attachedHere)
	if env == nil {
		runtime.UnlockOSThread()
		return thread{}, statusError
	}
	return thread{env: env, attachedHere: attachedHere}, statusOK
}

func (t thread) leave() {
	C.ravenpass_bridge_leave(t.env, t.attachedHere)
	runtime.UnlockOSThread()
}

// payload is the part of a Bridge result after its status byte, in C memory.
type payload struct {
	data   *C.uint8_t
	length C.size_t
}

// take copies the payload into Go memory and wipes and frees C's copy.
func (p payload) take() []byte {
	if p.data == nil {
		return nil
	}
	defer C.ravenpass_bridge_release(p.data, p.length)
	return C.GoBytes(unsafe.Pointer(p.data), C.int(p.length))
}

func createKey(alias string, presence bool) (status, []byte) {
	t, s := enter()
	if s != statusOK {
		return s, nil
	}
	defer t.leave()
	var out payload
	s = status(C.ravenpass_bridge_create_key(t.env, stringData(alias), C.jsize(len(alias)), C.bool(presence),
		&out.data, &out.length))
	return s, out.take()
}

func agree(alias string, peer []byte) (status, []byte) {
	t, s := enter()
	if s != statusOK {
		return s, nil
	}
	defer t.leave()
	var out payload
	s = status(C.ravenpass_bridge_agree(t.env, stringData(alias), C.jsize(len(alias)),
		unsafe.Pointer(unsafe.SliceData(peer)), C.jsize(len(peer)), &out.data, &out.length))
	return s, out.take()
}

func decrypt(alias string, ciphertext []byte, reason string) (status, []byte) {
	t, s := enter()
	if s != statusOK {
		return s, nil
	}
	defer t.leave()
	var out payload
	s = status(C.ravenpass_bridge_decrypt(t.env, stringData(alias), C.jsize(len(alias)),
		unsafe.Pointer(unsafe.SliceData(ciphertext)), C.jsize(len(ciphertext)), stringData(reason),
		C.jsize(len(reason)), &out.data, &out.length))
	return s, out.take()
}

func ownerAvailable() bool {
	t, s := enter()
	if s != statusOK {
		return false
	}
	defer t.leave()
	return bool(C.ravenpass_bridge_owner_available(t.env))
}

func authenticate(reason string) status {
	t, s := enter()
	if s != statusOK {
		return s
	}
	defer t.leave()
	return status(C.ravenpass_bridge_authenticate(t.env, stringData(reason), C.jsize(len(reason))))
}

func cancelAuthentication() {
	t, s := enter()
	if s != statusOK {
		return
	}
	defer t.leave()
	C.ravenpass_bridge_cancel_authentication(t.env)
}

func writeText(text string) int64 {
	t, s := enter()
	if s != statusOK {
		return -1
	}
	defer t.leave()
	return int64(C.ravenpass_bridge_write_text(t.env, stringData(text), C.jsize(len(text))))
}

func changeCount() int64 {
	t, s := enter()
	if s != statusOK {
		return -1
	}
	defer t.leave()
	return int64(C.ravenpass_bridge_change_count(t.env))
}

func clearClipboard() {
	t, s := enter()
	if s != statusOK {
		return
	}
	defer t.leave()
	C.ravenpass_bridge_clear_clipboard(t.env)
}

func writeScan(content []byte, mediaType string) int64 {
	t, s := enter()
	if s != statusOK {
		return -1
	}
	defer t.leave()
	return int64(C.ravenpass_bridge_write_scan(t.env, unsafe.Pointer(unsafe.SliceData(content)), C.jsize(len(content)),
		stringData(mediaType), C.jsize(len(mediaType))))
}

func allowScreenshots(allowed bool) {
	t, s := enter()
	if s != statusOK {
		return
	}
	defer t.leave()
	C.ravenpass_bridge_allow_screenshots(t.env, C.bool(allowed))
}

func printPage(job string, page []byte) status {
	t, s := enter()
	if s != statusOK {
		return s
	}
	defer t.leave()
	return status(C.ravenpass_bridge_print(t.env, stringData(job), C.jsize(len(job)),
		unsafe.Pointer(unsafe.SliceData(page)), C.jsize(len(page))))
}

func pickDocument(create bool, name string) (status, []byte) {
	t, s := enter()
	if s != statusOK {
		return s, nil
	}
	defer t.leave()
	var out payload
	s = status(C.ravenpass_bridge_pick_document(t.env, C.bool(create), stringData(name), C.jsize(len(name)),
		&out.data, &out.length))
	return s, out.take()
}

func createDocument(name, mediaType string) (status, []byte) {
	t, s := enter()
	if s != statusOK {
		return s, nil
	}
	defer t.leave()
	var out payload
	s = status(C.ravenpass_bridge_create_document(t.env, stringData(name), C.jsize(len(name)),
		stringData(mediaType), C.jsize(len(mediaType)), &out.data, &out.length))
	return s, out.take()
}

func pickPhoto() (status, []byte) {
	t, s := enter()
	if s != statusOK {
		return s, nil
	}
	defer t.leave()
	var out payload
	s = status(C.ravenpass_bridge_pick_photo(t.env, &out.data, &out.length))
	return s, out.take()
}

func readDocument(address string, limit int64) (status, []byte) {
	t, s := enter()
	if s != statusOK {
		return s, nil
	}
	defer t.leave()
	var out payload
	s = status(C.ravenpass_bridge_read_document(t.env, stringData(address), C.jsize(len(address)), C.int64_t(limit),
		&out.data, &out.length))
	return s, out.take()
}

func writeDocument(address string, data []byte) status {
	t, s := enter()
	if s != statusOK {
		return s
	}
	defer t.leave()
	return status(C.ravenpass_bridge_write_document(t.env, stringData(address), C.jsize(len(address)),
		unsafe.Pointer(unsafe.SliceData(data)), C.jsize(len(data))))
}

func deleteDocument(address string) status {
	t, s := enter()
	if s != statusOK {
		return s
	}
	defer t.leave()
	return status(C.ravenpass_bridge_delete_document(t.env, stringData(address), C.jsize(len(address))))
}

func pickFolder() (status, []byte) {
	t, s := enter()
	if s != statusOK {
		return s, nil
	}
	defer t.leave()
	var out payload
	s = status(C.ravenpass_bridge_pick_folder(t.env, &out.data, &out.length))
	return s, out.take()
}

func createInFolder(folder, name, mediaType string) (status, []byte) {
	t, s := enter()
	if s != statusOK {
		return s, nil
	}
	defer t.leave()
	var out payload
	s = status(C.ravenpass_bridge_create_in_folder(t.env, stringData(folder), C.jsize(len(folder)),
		stringData(name), C.jsize(len(name)), stringData(mediaType), C.jsize(len(mediaType)),
		&out.data, &out.length))
	return s, out.take()
}

func autofillSelected() bool {
	t, s := enter()
	if s != statusOK {
		return false
	}
	defer t.leave()
	return bool(C.ravenpass_bridge_autofill_selected(t.env))
}

func passkeyProvider() int32 {
	t, s := enter()
	if s != statusOK {
		return -1
	}
	defer t.leave()
	return int32(C.ravenpass_bridge_passkey_provider(t.env))
}

func selectAutofill() status {
	t, s := enter()
	if s != statusOK {
		return s
	}
	defer t.leave()
	return status(C.ravenpass_bridge_select_autofill(t.env))
}

func preferredLanguages() []byte {
	t, s := enter()
	if s != statusOK {
		return nil
	}
	defer t.leave()
	var out payload
	out.data = C.ravenpass_bridge_preferred_languages(t.env, &out.length)
	return out.take()
}

func setLanguage(tag string) {
	t, s := enter()
	if s != statusOK {
		return
	}
	defer t.leave()
	C.ravenpass_bridge_set_language(t.env, stringData(tag), C.jsize(len(tag)))
}

func setAppearance(appearance string) {
	t, s := enter()
	if s != statusOK {
		return
	}
	defer t.leave()
	C.ravenpass_bridge_set_appearance(t.env, stringData(appearance), C.jsize(len(appearance)))
}

func systemVersion() []byte {
	t, s := enter()
	if s != statusOK {
		return nil
	}
	defer t.leave()
	var out payload
	out.data = C.ravenpass_bridge_system_version(t.env, &out.length)
	return out.take()
}

func appName(pkg string) []byte {
	t, s := enter()
	if s != statusOK {
		return nil
	}
	defer t.leave()
	var out payload
	out.data = C.ravenpass_bridge_app_name(t.env, stringData(pkg), C.jsize(len(pkg)), &out.length)
	return out.take()
}

func thirdPartyNotices() (status, []byte) {
	t, s := enter()
	if s != statusOK {
		return s, nil
	}
	defer t.leave()
	var out payload
	s = status(C.ravenpass_bridge_third_party_notices(t.env, &out.data, &out.length))
	return s, out.take()
}

func stringData(value string) unsafe.Pointer { return unsafe.Pointer(unsafe.StringData(value)) }
