import Foundation
import Darwin

// Native diagnostics are append-only, metadata-only, and intentionally kept
// separate from the Computer Use protocol response. They are for tracing
// nondeterministic window failures and must never contain image bytes, secrets,
// headers, or raw provider/helper errors.
enum NativeDiagnostics {
    // Diagnostics are opt-in. The old unconditional /tmp file could retain
    // window metadata indefinitely and violated the runtime privacy boundary.
    // An explicit path is accepted only when it is absolute and its parent is
    // an owner-only directory; ordinary builds emit no native diagnostics.
    private static let path: String? = {
        guard let raw = ProcessInfo.processInfo.environment["GO_E2E_COMPUTER_DIAGNOSTICS_PATH"],
              raw.hasPrefix("/"), !raw.contains("\0") else { return nil }
        let url = URL(fileURLWithPath: raw)
        let parent = url.deletingLastPathComponent().path
        guard let attributes = try? FileManager.default.attributesOfItem(atPath: parent),
              let permissions = attributes[.posixPermissions] as? NSNumber,
              permissions.intValue & 0o077 == 0,
              let owner = attributes[.ownerAccountID] as? NSNumber,
              owner.uintValue == getuid() else { return nil }
        return url.path
    }()
    private static let lock = NSLock()
    private static let formatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    static func append(_ fields: [String: Any]) {
        guard let path else { return }
        lock.lock()
        defer { lock.unlock() }
        var event = fields
        event["schema_version"] = "computer-use-diagnostic.v1"
        event["timestamp"] = formatter.string(from: Date())
        guard JSONSerialization.isValidJSONObject(event),
              let data = try? JSONSerialization.data(withJSONObject: event),
              var line = String(data: data, encoding: .utf8) else { return }
        line.append("\n")

        if let handle = FileHandle(forWritingAtPath: path) {
            handle.seekToEndOfFile()
            handle.write(line.data(using: .utf8) ?? Data())
            try? handle.close()
        } else {
            FileManager.default.createFile(atPath: path, contents: line.data(using: .utf8), attributes: [.posixPermissions: 0o600])
        }
    }
}
