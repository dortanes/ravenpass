import CryptoKit
import Foundation
import LocalAuthentication
import Security

private let secretSize = 32

// Result codes shared with enclave_darwin_arm64.go.
private let succeeded: Int32 = 0
private let failed: Int32 = 1
private let canceled: Int32 = 2
private let notAuthenticated: Int32 = 3
private let ownerUnavailable: Int32 = 4
private let interactionRequired: Int32 = 5

private struct AccessControlUnavailable: Error {}

private struct KeychainFailure: Error {
    let status: OSStatus
}

private let labelSize = 16

// Raw values match the keychain constants in keychain_signed.go and keychain_adhoc.go.
private enum Keychain: Int32 {
    // The app's default access group, the application identifier: code signed by another team or for another app
    // cannot read the item. Writing to it needs that entitlement, which an ad-hoc signature lacks.
    case dataProtection = 0
    // The login keychain, where only the item's access list guards it; for ad-hoc builds only.
    case login = 1
}

private func hex(_ bytes: Data) -> String {
    bytes.map { String(format: "%02x", $0) }.joined()
}

// The account names the salt the key serves, so a key in use for a salt can find and remove the salt's other keys.
private func keyAccount(salt: Data, label: Data) -> String {
    "\(hex(salt)).\(hex(label))"
}

private func keyQuery(_ keychain: Keychain, _ service: String, account: String? = nil) -> [String: Any] {
    var query: [String: Any] = [
        kSecClass as String: kSecClassGenericPassword,
        kSecAttrService as String: service,
    ]
    if keychain == .dataProtection {
        query[kSecUseDataProtectionKeychain as String] = true
    }
    if let account {
        query[kSecAttrAccount as String] = account
    }
    return query
}

private func randomLabel() throws -> Data {
    var label = Data(count: labelSize)
    let status = label.withUnsafeMutableBytes { bytes in
        SecRandomCopyBytes(kSecRandomDefault, labelSize, bytes.baseAddress!)
    }
    guard status == errSecSuccess else {
        throw KeychainFailure(status: status)
    }
    return label
}

// Stores the key for salt under a new label and returns the label; the salt's earlier keys stay until this one is used.
private func storeKey(
    _ key: SecureEnclave.P256.KeyAgreement.PrivateKey, in keychain: Keychain, service: String, salt: Data
) throws -> Data {
    let label = try randomLabel()
    var item = keyQuery(keychain, service, account: keyAccount(salt: salt, label: label))
    item[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlockedThisDeviceOnly
    item[kSecAttrSynchronizable as String] = false
    item[kSecValueData as String] = key.dataRepresentation
    let status = SecItemAdd(item as CFDictionary, nil)
    guard status == errSecSuccess else {
        throw KeychainFailure(status: status)
    }
    return label
}

// Runs once the key labelled label is used, so a replacement whose record was never stored, as in a recovery key
// change another device's save refused, leaves the record's key in place. A key left behind only stays unused.
private func removeOtherKeys(in keychain: Keychain, service: String, salt: Data, keeping label: Data) {
    var query = keyQuery(keychain, service)
    query[kSecMatchLimit as String] = kSecMatchLimitAll
    query[kSecReturnAttributes as String] = true
    var result: CFTypeRef?
    guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
        let items = result as? [[String: Any]]
    else {
        return
    }
    let prefix = "\(hex(salt))."
    let account = keyAccount(salt: salt, label: label)
    for item in items {
        guard let other = item[kSecAttrAccount as String] as? String,
            other.hasPrefix(prefix), other != account
        else {
            continue
        }
        SecItemDelete(keyQuery(keychain, service, account: other) as CFDictionary)
    }
}

private func loadKey(in keychain: Keychain, service: String, salt: Data, label: Data) throws -> Data {
    var query = keyQuery(keychain, service, account: keyAccount(salt: salt, label: label))
    query[kSecMatchLimit as String] = kSecMatchLimitOne
    query[kSecReturnData as String] = true
    var result: CFTypeRef?
    let status = SecItemCopyMatching(query as CFDictionary, &result)
    guard status == errSecSuccess, let data = result as? Data else {
        throw KeychainFailure(status: status)
    }
    return data
}

// Raw values match policy in secureenclave.go.
private enum Policy: Int32 {
    case none = 0
    case userPresence = 1

    var sharedInfo: Data {
        switch self {
        case .none: return Data("ravenpass/pin/v1/secure-enclave".utf8)
        case .userPresence: return Data("ravenpass/unlock/v1/secure-enclave".utf8)
        }
    }

    // Each policy keeps its keys apart, as a PIN and device authentication share the vault's salt.
    var keychainService: String {
        switch self {
        case .none: return "com.dortanes.ravenpass.pin-key"
        case .userPresence: return "com.dortanes.ravenpass.unlock-key"
        }
    }

    func makeKey() throws -> SecureEnclave.P256.KeyAgreement.PrivateKey {
        switch self {
        case .none:
            return try SecureEnclave.P256.KeyAgreement.PrivateKey()
        case .userPresence:
            // Without privateKeyUsage, key agreement fails with LAError -1009 (measured on macOS 27).
            var error: Unmanaged<CFError>?
            guard let access = SecAccessControlCreateWithFlags(
                nil, kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
                [.privateKeyUsage, .userPresence], &error)
            else {
                throw (error?.takeRetainedValue() as Error?) ?? AccessControlUnavailable()
            }
            return try SecureEnclave.P256.KeyAgreement.PrivateKey(accessControl: access)
        }
    }
}

// The shared secret and derived key stay in CryptoKit storage, which zeroes itself on release.
private func derive(
    _ shared: SharedSecret, _ salt: Data, _ policy: Policy, _ secret: UnsafeMutablePointer<UInt8>
) {
    let derived = shared.hkdfDerivedSymmetricKey(
        using: SHA256.self, salt: salt, sharedInfo: policy.sharedInfo, outputByteCount: secretSize)
    derived.withUnsafeBytes { bytes in
        UnsafeMutableRawBufferPointer(start: secret, count: secretSize).copyMemory(from: bytes)
    }
}

private func outcome(of error: Error) -> Int32 {
    let error = error as NSError
    guard error.domain == LAErrorDomain else {
        return failed
    }
    switch error.code {
    case LAError.Code.userCancel.rawValue, LAError.Code.systemCancel.rawValue,
        LAError.Code.appCancel.rawValue, LAError.Code.userFallback.rawValue:
        return canceled
    case LAError.Code.authenticationFailed.rawValue:
        return notAuthenticated
    case LAError.Code.notInteractive.rawValue:
        return interactionRequired
    default:
        return ownerUnavailable
    }
}

@_cdecl("ravenpass_enclave_create")
public func ravenpassEnclaveCreate(
    _ keychain: Int32, _ policy: Int32,
    _ salt: UnsafePointer<UInt8>, _ saltLength: Int,
    _ boundKey: UnsafeMutablePointer<UInt8>, _ boundKeyCapacity: Int,
    _ boundKeyLength: UnsafeMutablePointer<Int>,
    _ peerKey: UnsafeMutablePointer<UInt8>, _ peerKeyCapacity: Int,
    _ peerKeyLength: UnsafeMutablePointer<Int>,
    _ secret: UnsafeMutablePointer<UInt8>
) -> Int32 {
    guard let keychain = Keychain(rawValue: keychain), let policy = Policy(rawValue: policy) else {
        return failed
    }
    do {
        let saltData = Data(bytes: salt, count: saltLength)
        let key = try policy.makeKey()
        let peer = P256.KeyAgreement.PrivateKey()
        let peerData = peer.publicKey.x963Representation
        if labelSize > boundKeyCapacity || peerData.count > peerKeyCapacity {
            return failed
        }
        // Agreement from the software key's side never uses the enclave key and never prompts.
        let shared = try peer.sharedSecretFromKeyAgreement(with: key.publicKey)
        let label = try storeKey(key, in: keychain, service: policy.keychainService, salt: saltData)
        derive(shared, saltData, policy, secret)
        label.copyBytes(to: boundKey, count: label.count)
        peerData.copyBytes(to: peerKey, count: peerData.count)
        boundKeyLength.pointee = label.count
        peerKeyLength.pointee = peerData.count
        return succeeded
    } catch {
        return failed
    }
}

// Without a reason no prompt may show, and a key that needs its owner fails with interactionRequired.
@_cdecl("ravenpass_enclave_derive")
public func ravenpassEnclaveDerive(
    _ keychain: Int32, _ policy: Int32,
    _ boundKey: UnsafePointer<UInt8>, _ boundKeyLength: Int,
    _ peerKey: UnsafePointer<UInt8>, _ peerKeyLength: Int,
    _ salt: UnsafePointer<UInt8>, _ saltLength: Int,
    _ reason: UnsafePointer<UInt8>?, _ reasonLength: Int,
    _ secret: UnsafeMutablePointer<UInt8>
) -> Int32 {
    guard let keychain = Keychain(rawValue: keychain), let policy = Policy(rawValue: policy) else {
        return failed
    }
    let context = LAContext()
    if let reason, reasonLength > 0,
        let text = String(bytes: UnsafeBufferPointer(start: reason, count: reasonLength), encoding: .utf8)
    {
        context.localizedReason = text
    } else {
        context.interactionNotAllowed = true
    }
    do {
        let saltData = Data(bytes: salt, count: saltLength)
        let label = Data(bytes: boundKey, count: boundKeyLength)
        let representation = try loadKey(
            in: keychain, service: policy.keychainService, salt: saltData, label: label)
        let key = try SecureEnclave.P256.KeyAgreement.PrivateKey(
            dataRepresentation: representation, authenticationContext: context)
        let peer = try P256.KeyAgreement.PublicKey(
            x963Representation: Data(bytes: peerKey, count: peerKeyLength))
        let shared = try key.sharedSecretFromKeyAgreement(with: peer)
        derive(shared, saltData, policy, secret)
        removeOtherKeys(in: keychain, service: policy.keychainService, salt: saltData, keeping: label)
        return succeeded
    } catch {
        return outcome(of: error)
    }
}
