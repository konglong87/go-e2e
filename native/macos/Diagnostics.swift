import Foundation

// Native diagnostics are append-only, metadata-only, and intentionally kept
// separate from the Computer Use protocol response. They are for tracing
// nondeterministic window failures and must never contain image bytes, secrets,
// headers, or raw provider/helper errors.
enum NativeDiagnostics {
    static let path = "/tmp/swift-mismatch.log"
    private static let lock = NSLock()
    private static let formatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    static func append(_ fields: [String: Any]) {
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
