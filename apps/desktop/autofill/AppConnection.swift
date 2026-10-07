import AppKit
import AuthenticationServices
import notify

/// The kind of value the system asks the extension to hand over.
enum SecretKind {
    case password
    case code
}

/// A value the app released for one credential.
enum Secret {
    case password(user: String, password: String)
    case code(String)
}

/// Why the app did not give what it was asked.
enum AppError: Error {
    /// No Ravenpass answers at the socket: it is not running, or the process there is not Ravenpass.
    case unreachable
    /// An update replaced the running Ravenpass on disk; it answers once started again.
    case outdated
    case locked
    case notFound
    /// The credential is saved for other websites.
    case noMatch
    /// The app could not add the website to the credential.
    case notAdded
    case noCode
    /// The owner declined unlocking or confirming it's them, or did not answer while the app waited.
    case declined
    /// The relying party requires confirming it's the owner, and the vault offers no way to.
    case unverifiable
    /// The vault holds a passkey the relying party excluded.
    case excluded
    /// The relying party accepts no kind of passkey the vault creates.
    case unsupported
    case canceled
    case failed
}

/// A service identifier the system passed, as the app reads it: a URL or a domain name.
struct ServiceIdentifier: Encodable {
    let kind: String
    let value: String

    /// The identifier as the app reads it, or nil for a kind the app does not match.
    init?(_ identifier: ASCredentialServiceIdentifier) {
        switch identifier.type {
        case .URL: kind = "url"
        case .domain: kind = "domain"
        default: return nil
        }
        value = identifier.identifier
    }
}

/// A listed credential; matches is false for one that fills only once the owner adds the site.
struct Suggestion: Codable {
    let id: String
    let label: String
    let account: String
    let site: String
    let matches: Bool
    /// Absent for a credential without tags and from an app that predates them.
    let tags: [String]?
}

/// site is the site of the most specific service.
struct Listed {
    let site: String
    let scope: String
    let suggestions: [Suggestion]
}

/// The passkeys the app lists for a relying party, and the relying party's site.
struct PasskeyList {
    let site: String
    let passkeys: [Passkey]
}

/// A site's icon as a base64 PNG, empty for none, and its main colour as "#rrggbb".
struct SiteIcon {
    let image: String
    let tint: String
}

/// The languages Ravenpass offers, the one in use, and whether the owner chose it.
struct LanguageSettings {
    let languages: [String]
    let language: String
    let chosen: Bool
}

/// One credential's value to release for the websites the system named, most specific first.
struct Release {
    let kind: SecretKind
    let id: String
    let services: [ServiceIdentifier]
}

extension Release {
    /// The release a system request names, or nil for a request that names no Ravenpass credential.
    init?(_ request: ASCredentialRequest) {
        let identity = request.credentialIdentity
        guard let id = identity.recordIdentifier, let service = ServiceIdentifier(identity.serviceIdentifier) else {
            return nil
        }
        switch request.type {
        case .password: kind = .password
        case .oneTimeCode: kind = .code
        default: return nil
        }
        self.id = id
        services = [service]
    }
}

/// The running app, framed as apps/desktop/internal/autofillbridge/protocol.go lays out; calls block.
final class AppConnection: @unchecked Sendable {
    private static let group = "group.com.dortanes.ravenpass"
    private static let socketName = "fill"
    /// The Darwin notification the app posts once its socket listens.
    private static let listeningNotice = group + ".listening"
    private static let maxAnswerBytes: UInt32 = 1 << 20
    private static let exchangeTimeout: TimeInterval = 5
    /// Added to the wait the app announces, for the app to write its answer.
    private static let answerMargin: TimeInterval = 10
    private static let launchTimeout: TimeInterval = 20

    private let lock = NSLock()
    private var sockets: Set<Int32> = []
    private var canceled = false

    /// Reports whether the vault is open, calling launching and starting Ravenpass when it is not running.
    func status(launching: () -> Void) throws -> Bool {
        do {
            return try ask(Request(op: "status")).open
        } catch AppError.unreachable {}
        let listening = ListeningSignal(Self.listeningNotice)
        launching()
        launch()
        let deadline = DispatchTime.now() + Self.launchTimeout
        var waiting = true
        while true {
            do {
                return try ask(Request(op: "status")).open
            } catch AppError.unreachable where waiting {
                waiting = listening.wait(until: deadline)
            }
        }
    }

    func unlock() throws -> Bool {
        try ask(Request(op: "unlock")).open
    }

    /// An empty query lists what matches the services, else the vault's credentials the query finds.
    func search(_ query: String, services: [ServiceIdentifier], codes: Bool) throws -> Listed {
        let answer = try ask(Request(op: "search", services: services, codes: codes, query: query))
        return Listed(site: answer.site, scope: answer.scope, suggestions: answer.suggestions)
    }

    func icon(_ site: String) throws -> SiteIcon {
        let answer = try ask(Request(op: "icon", site: site))
        return SiteIcon(image: answer.image, tint: answer.tint)
    }

    func language() throws -> LanguageSettings {
        let answer = try ask(Request(op: "language"))
        return LanguageSettings(languages: answer.languages, language: answer.language, chosen: answer.chosen)
    }

    /// "system", "light" or "dark".
    func appearance() throws -> String {
        try ask(Request(op: "appearance")).appearance
    }

    /// The app releases the value only once the credential matches one of the services.
    func release(_ wanted: Release) throws -> Secret {
        switch wanted.kind {
        case .password:
            let answer = try ask(Request(op: "password", services: wanted.services, id: wanted.id))
            return .password(user: answer.user, password: answer.password)
        case .code:
            return .code(try ask(Request(op: "code", services: wanted.services, id: wanted.id)).code)
        }
    }

    /// Adds the most specific website of the release's services to its credential.
    func addSite(for wanted: Release) throws {
        _ = try ask(Request(op: "add-site", services: wanted.services, id: wanted.id))
    }

    func passkeys(for query: PasskeyQuery) throws -> PasskeyList {
        let answer = try ask(Request(op: "passkeys", rpID: query.relyingParty, allowed: query.allowed))
        return PasskeyList(site: answer.site, passkeys: answer.passkeys)
    }

    func sign(_ signIn: PasskeySignIn) throws -> PasskeyAssertion {
        let answer = try ask(Request(
            op: "passkey-sign", id: signIn.id, rpID: signIn.relyingParty, clientDataHash: signIn.clientDataHash,
            credentialID: signIn.credentialID, verification: signIn.verification
        ))
        return PasskeyAssertion(
            credentialID: answer.credentialID, authenticatorData: answer.authenticatorData,
            signature: answer.signature, userHandle: answer.userHandle
        )
    }

    func create(_ registration: PasskeyRegistration) throws -> CreatedPasskey {
        let answer = try ask(Request(
            op: "passkey-create", rpID: registration.relyingParty, clientDataHash: registration.clientDataHash,
            user: User(handle: registration.userHandle, name: registration.userName),
            algorithms: registration.algorithms, excluded: registration.excluded,
            verification: registration.verification
        ))
        return CreatedPasskey(credentialID: answer.credentialID, attestationObject: answer.attestationObject)
    }

    /// Ends the exchanges under way and refuses later ones.
    func cancel() {
        lock.lock()
        defer { lock.unlock() }
        canceled = true
        for socket in sockets {
            shutdown(socket, SHUT_RDWR)
        }
    }

    private func ask(_ request: Request) throws -> Answer {
        let body: Data
        do {
            body = try JSONEncoder().encode(request)
        } catch {
            throw AppError.failed
        }
        let answer = try exchange(body)
        switch answer.error {
        case nil: return answer
        case "locked": throw AppError.locked
        case "not-found": throw AppError.notFound
        case "no-match": throw AppError.noMatch
        case "no-code": throw AppError.noCode
        case "declined": throw AppError.declined
        case "unverifiable": throw AppError.unverifiable
        case "excluded": throw AppError.excluded
        case "unsupported": throw AppError.unsupported
        default: throw AppError.failed
        }
    }

    private func exchange(_ body: Data) throws -> Answer {
        let socket = try openSocket()
        defer { closeSocket(socket) }
        guard connectToApp(socket) else {
            throw AppError.unreachable
        }
        switch CodeSignature.peer(of: socket) {
        case .ravenpass: break
        case .outdated: throw AppError.outdated
        case .other: throw AppError.unreachable
        }
        setTimeout(socket, option: SO_SNDTIMEO, seconds: Self.exchangeTimeout)
        setTimeout(socket, option: SO_RCVTIMEO, seconds: Self.exchangeTimeout)
        var length = UInt32(body.count).bigEndian
        guard send(socket, Data(bytes: &length, count: 4) + body) else {
            throw AppError.failed
        }
        let first = try receiveAnswer(socket)
        guard let wait = first.wait else {
            return first
        }
        setTimeout(socket, option: SO_RCVTIMEO, seconds: TimeInterval(wait) + Self.answerMargin)
        let answer = try receiveAnswer(socket)
        guard answer.wait == nil else {
            throw AppError.failed
        }
        return answer
    }

    private func receiveAnswer(_ socket: Int32) throws -> Answer {
        guard let header = receive(socket, count: 4) else {
            throw AppError.failed
        }
        let size = header.withUnsafeBytes { UInt32(bigEndian: $0.loadUnaligned(as: UInt32.self)) }
        guard size > 0, size <= Self.maxAnswerBytes, var message = receive(socket, count: Int(size)) else {
            throw AppError.failed
        }
        defer { message.resetBytes(in: message.indices) }
        do {
            return try JSONDecoder().decode(Answer.self, from: message)
        } catch {
            throw AppError.failed
        }
    }

    private func openSocket() throws -> Int32 {
        lock.lock()
        defer { lock.unlock() }
        guard !canceled else { throw AppError.canceled }
        let created = Darwin.socket(AF_UNIX, SOCK_STREAM, 0)
        guard created >= 0 else { throw AppError.unreachable }
        var on: Int32 = 1
        setsockopt(created, SOL_SOCKET, SO_NOSIGPIPE, &on, socklen_t(MemoryLayout<Int32>.size))
        sockets.insert(created)
        return created
    }

    private func closeSocket(_ closing: Int32) {
        lock.lock()
        defer { lock.unlock() }
        sockets.remove(closing)
        Darwin.close(closing)
    }

    private func connectToApp(_ socket: Int32) -> Bool {
        guard let container = FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: Self.group) else {
            return false
        }
        let path = Array(container.appendingPathComponent(Self.socketName).path(percentEncoded: false).utf8)
        var address = sockaddr_un()
        guard path.count < MemoryLayout.size(ofValue: address.sun_path) else { return false }
        address.sun_family = sa_family_t(AF_UNIX)
        address.sun_len = UInt8(MemoryLayout<sockaddr_un>.size)
        withUnsafeMutableBytes(of: &address.sun_path) { $0.copyBytes(from: path) }
        let result = withUnsafePointer(to: &address) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                Darwin.connect(socket, $0, socklen_t(MemoryLayout<sockaddr_un>.size))
            }
        }
        return result == 0
    }

    private func setTimeout(_ socket: Int32, option: Int32, seconds: TimeInterval) {
        var interval = timeval(tv_sec: Int(seconds), tv_usec: 0)
        setsockopt(socket, SOL_SOCKET, option, &interval, socklen_t(MemoryLayout<timeval>.size))
    }

    private func send(_ socket: Int32, _ data: Data) -> Bool {
        data.withUnsafeBytes { buffer in
            var offset = 0
            while offset < buffer.count {
                let sent = Darwin.write(socket, buffer.baseAddress! + offset, buffer.count - offset)
                if sent < 0 && errno == EINTR {
                    continue
                }
                if sent <= 0 {
                    return false
                }
                offset += sent
            }
            return true
        }
    }

    private func receive(_ socket: Int32, count: Int) -> Data? {
        var data = Data(count: count)
        let complete = data.withUnsafeMutableBytes { buffer in
            var offset = 0
            while offset < count {
                let read = Darwin.read(socket, buffer.baseAddress! + offset, count - offset)
                if read < 0 && errno == EINTR {
                    continue
                }
                if read <= 0 {
                    return false
                }
                offset += read
            }
            return true
        }
        guard complete else {
            data.resetBytes(in: data.indices)
            return nil
        }
        return data
    }

    /// Starts the app that contains the extension, in the background.
    private func launch() {
        let app = Bundle.main.bundleURL
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
        let configuration = NSWorkspace.OpenConfiguration()
        configuration.activates = false
        configuration.addsToRecentItems = false
        NSWorkspace.shared.openApplication(at: app, configuration: configuration, completionHandler: nil)
    }
}

/// Signals each Darwin notification of one name from the moment it is created.
private final class ListeningSignal: @unchecked Sendable {
    private let posted = DispatchSemaphore(value: 0)
    private var token: Int32 = 0
    private var registered = false

    init(_ name: String) {
        let posted = self.posted
        registered = notify_register_dispatch(name, &token, DispatchQueue.global()) { _ in
            posted.signal()
        } == NOTIFY_STATUS_OK
    }

    deinit {
        if registered {
            notify_cancel(token)
        }
    }

    /// Reports whether the notification arrived before deadline.
    func wait(until deadline: DispatchTime) -> Bool {
        posted.wait(timeout: deadline) == .success
    }
}

/// What the extension asks the app. Data travels as Base64.
private struct Request: Encodable {
    let op: String
    var services: [ServiceIdentifier]? = nil
    var codes: Bool? = nil
    var id: String? = nil
    var query: String? = nil
    var site: String? = nil
    var rpID: String? = nil
    var allowed: [Data]? = nil
    var clientDataHash: Data? = nil
    var credentialID: Data? = nil
    var user: User? = nil
    var algorithms: [Int]? = nil
    var excluded: [Data]? = nil
    var verification: Verification? = nil
}

/// The account a relying party creates a passkey for.
private struct User: Encodable {
    let handle: Data
    let name: String
}

/// A reply with wait alone announces the seconds the app may take to send the next one.
private struct Answer: Decodable {
    let wait: Int?
    let error: String?
    let site: String
    let open: Bool
    let scope: String
    let suggestions: [Suggestion]
    let user: String
    let password: String
    let code: String
    let passkeys: [Passkey]
    let credentialID: Data
    let authenticatorData: Data
    let signature: Data
    let userHandle: Data
    let attestationObject: Data
    let image: String
    let tint: String
    let languages: [String]
    let language: String
    let chosen: Bool
    let appearance: String

    private enum CodingKeys: String, CodingKey {
        case wait, error, site, open, scope, suggestions, user, password, code, passkeys
        case credentialID, authenticatorData, signature, userHandle, attestationObject
        case image, tint, languages, language, chosen, appearance
    }

    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        wait = try values.decodeIfPresent(Int.self, forKey: .wait)
        error = try values.decodeIfPresent(String.self, forKey: .error)
        site = try values.decodeIfPresent(String.self, forKey: .site) ?? ""
        open = try values.decodeIfPresent(Bool.self, forKey: .open) ?? false
        scope = try values.decodeIfPresent(String.self, forKey: .scope) ?? ""
        suggestions = try values.decodeIfPresent([Suggestion].self, forKey: .suggestions) ?? []
        user = try values.decodeIfPresent(String.self, forKey: .user) ?? ""
        password = try values.decodeIfPresent(String.self, forKey: .password) ?? ""
        code = try values.decodeIfPresent(String.self, forKey: .code) ?? ""
        passkeys = try values.decodeIfPresent([Passkey].self, forKey: .passkeys) ?? []
        credentialID = try values.decodeIfPresent(Data.self, forKey: .credentialID) ?? Data()
        authenticatorData = try values.decodeIfPresent(Data.self, forKey: .authenticatorData) ?? Data()
        signature = try values.decodeIfPresent(Data.self, forKey: .signature) ?? Data()
        userHandle = try values.decodeIfPresent(Data.self, forKey: .userHandle) ?? Data()
        attestationObject = try values.decodeIfPresent(Data.self, forKey: .attestationObject) ?? Data()
        image = try values.decodeIfPresent(String.self, forKey: .image) ?? ""
        tint = try values.decodeIfPresent(String.self, forKey: .tint) ?? ""
        languages = try values.decodeIfPresent([String].self, forKey: .languages) ?? []
        language = try values.decodeIfPresent(String.self, forKey: .language) ?? ""
        chosen = try values.decodeIfPresent(Bool.self, forKey: .chosen) ?? false
        appearance = try values.decodeIfPresent(String.self, forKey: .appearance) ?? ""
    }
}
