import Foundation
import Security

// Result codes shared with sandbox_darwin_arm64.go.
private let succeeded: Int32 = 0
private let failed: Int32 = 1
private let stale: Int32 = 2

private func string(_ bytes: UnsafePointer<UInt8>, _ length: Int) -> String {
    String(decoding: UnsafeBufferPointer(start: bytes, count: length), as: UTF8.self)
}

// Hands data to Go in memory from malloc, which the caller frees.
private func hand(_ data: Data, _ out: UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>, _ length: UnsafeMutablePointer<Int>) {
    let copy = UnsafeMutablePointer<UInt8>.allocate(capacity: max(data.count, 1))
    data.copyBytes(to: copy, count: data.count)
    out.pointee = copy
    length.pointee = data.count
}

// The process's own signature says whether it runs in the App Sandbox; the environment does not for an app Finder opens.
@_cdecl("ravenpass_sandboxed")
public func ravenpassSandboxed() -> Int32 {
    guard let task = SecTaskCreateFromSelf(nil),
        let value = SecTaskCopyValueForEntitlement(task, "com.apple.security.app-sandbox" as CFString, nil)
    else { return 0 }
    return (value as? Bool) == true ? 1 : 0
}

@_cdecl("ravenpass_bookmark_create")
public func ravenpassBookmarkCreate(
    _ path: UnsafePointer<UInt8>, _ pathLength: Int,
    _ bookmark: UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>, _ bookmarkLength: UnsafeMutablePointer<Int>
) -> Int32 {
    let folder = URL(fileURLWithPath: string(path, pathLength), isDirectory: true)
    guard let data = try? folder.bookmarkData(options: .withSecurityScope, includingResourceValuesForKeys: nil, relativeTo: nil)
    else { return failed }
    hand(data, bookmark, bookmarkLength)
    return succeeded
}

// Resolving starts access to the folder for the rest of the process; it is never stopped.
@_cdecl("ravenpass_bookmark_resolve")
public func ravenpassBookmarkResolve(
    _ bookmark: UnsafePointer<UInt8>, _ bookmarkLength: Int,
    _ path: UnsafeMutablePointer<UnsafeMutablePointer<UInt8>?>, _ pathLength: UnsafeMutablePointer<Int>
) -> Int32 {
    var isStale = false
    guard let folder = try? URL(
        resolvingBookmarkData: Data(bytes: bookmark, count: bookmarkLength), options: .withSecurityScope,
        relativeTo: nil, bookmarkDataIsStale: &isStale
    ), folder.startAccessingSecurityScopedResource()
    else { return failed }
    hand(Data(folder.path.utf8), path, pathLength)
    return isStale ? stale : succeeded
}

// The sandbox lets the app touch only a file the owner picked in a save panel, not its folder, so the data is staged in
// the item replacement directory the system grants for that file, private and flushed, then swapped in with its mode.
@_cdecl("ravenpass_file_replace")
public func ravenpassFileReplace(
    _ path: UnsafePointer<UInt8>, _ pathLength: Int, _ data: UnsafePointer<UInt8>, _ dataLength: Int
) -> Int32 {
    let files = FileManager.default
    let file = URL(fileURLWithPath: string(path, pathLength))
    do {
        let staging = try files.url(for: .itemReplacementDirectory, in: .userDomainMask, appropriateFor: file, create: true)
        defer { try? files.removeItem(at: staging) }
        let staged = staging.appendingPathComponent(file.lastPathComponent)
        guard files.createFile(atPath: staged.path, contents: nil, attributes: [.posixPermissions: 0o600]) else {
            return failed
        }
        let handle = try FileHandle(forWritingTo: staged)
        defer { try? handle.close() }
        try handle.write(contentsOf: UnsafeBufferPointer(start: data, count: dataLength))
        try handle.synchronize()
        if files.fileExists(atPath: file.path) {
            _ = try files.replaceItemAt(file, withItemAt: staged, backupItemName: nil, options: .usingNewMetadataOnly)
        } else {
            try files.moveItem(at: staged, to: file)
        }
        return succeeded
    } catch {
        return failed
    }
}
