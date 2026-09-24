import AppKit
import ApplicationServices
import CoreGraphics
import Foundation
import ImageIO
import UniformTypeIdentifiers

let protocolVersion = "computer-use.v1"
let maxFrameBytes = 8 * 1024 * 1024
var paused = false
var stopped = false

struct Envelope: Codable {
    let protocolVersion: String
    let requestID: String
    let sessionID: String
    let actionID: String?
    let deadline: String?
    let command: String
    let payload: JSONValue?

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol_version"
        case requestID = "request_id"
        case sessionID = "session_id"
        case actionID = "action_id"
        case deadline
        case command
        case payload
    }
}

enum JSONValue: Codable {
    case object([String: JSONValue])
    case array([JSONValue])
    case string(String)
    case number(Double)
    case bool(Bool)
    case null

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() { self = .null; return }
        if let value = try? container.decode([String: JSONValue].self) { self = .object(value); return }
        if let value = try? container.decode([JSONValue].self) { self = .array(value); return }
        if let value = try? container.decode(String.self) { self = .string(value); return }
        if let value = try? container.decode(Double.self) { self = .number(value); return }
        if let value = try? container.decode(Bool.self) { self = .bool(value); return }
        throw DecodingError.dataCorruptedError(in: container, debugDescription: "unsupported JSON value")
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .object(let value): try container.encode(value)
        case .array(let value): try container.encode(value)
        case .string(let value): try container.encode(value)
        case .number(let value): try container.encode(value)
        case .bool(let value): try container.encode(value)
        case .null: try container.encodeNil()
        }
    }

    func string(_ key: String) -> String? {
        guard case .object(let object) = self, case .string(let value) = object[key] else { return nil }
        return value
    }

    func number(_ key: String) -> Int? {
        guard case .object(let object) = self, case .number(let value) = object[key] else { return nil }
        return Int(value)
    }
}

struct ResponseEnvelope: Codable {
    let protocolVersion: String
    let requestID: String
    let sessionID: String
    let actionID: String
    let deadline: String
    let command: String
    let payload: JSONValue

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol_version"
        case requestID = "request_id"
        case sessionID = "session_id"
        case actionID = "action_id"
        case deadline
        case command
        case payload
    }
}

func write(_ response: ResponseEnvelope) {
    let encoder = JSONEncoder()
    guard let body = try? encoder.encode(response), body.count <= maxFrameBytes else { return }
    var length = UInt32(body.count).bigEndian
    let prefix = withUnsafeBytes(of: &length) { Data($0) }
    FileHandle.standardOutput.write(prefix)
    FileHandle.standardOutput.write(body)
}

func readFrame() -> Data? {
    guard let prefix = try? FileHandle.standardInput.read(upToCount: 4), prefix.count == 4 else { return nil }
    let length = (UInt32(prefix[0]) << 24) | (UInt32(prefix[1]) << 16) | (UInt32(prefix[2]) << 8) | UInt32(prefix[3])
    guard length > 0 && length <= maxFrameBytes else { return nil }
    guard let body = try? FileHandle.standardInput.read(upToCount: Int(length)), body.count == Int(length) else { return nil }
    return body
}

func response(for request: Envelope, ok: Bool, outcome: String, payload: JSONValue? = nil, errorCode: String? = nil, errorMessage: String? = nil) {
    var result: [String: JSONValue] = ["ok": .bool(ok), "outcome": .string(outcome)]
    if let errorCode { result["error_code"] = .string(errorCode) }
    if let errorMessage { result["error_message"] = .string(errorMessage) }
    if let payload { result["result"] = payload }
    let responsePayload = JSONValue.object(result)
    write(ResponseEnvelope(protocolVersion: protocolVersion, requestID: request.requestID, sessionID: request.sessionID, actionID: request.actionID ?? request.requestID, deadline: request.deadline ?? ISO8601DateFormatter().string(from: Date().addingTimeInterval(10)), command: request.command, payload: responsePayload))
}

func readiness() -> JSONValue {
    let capture = CGPreflightScreenCaptureAccess()
    let accessibility = AXIsProcessTrusted()
    return .object([
        "capture_readiness": .string(capture ? "ready" : "permission_required"),
        "input_readiness": .string(accessibility ? "ready" : "permission_required"),
        "permission_state": .string(capture && accessibility ? "approved" : "required"),
        "focus_state": .string(NSWorkspace.shared.frontmostApplication == nil ? "unavailable" : "focused"),
        "platform": .string("macos"),
        "backend": .string("native_host"),
        "image_supported": .bool(true),
        "supports_pause": .bool(true),
        "supports_stop": .bool(true)
    ])
}

func pngData(_ image: CGImage) -> Data? {
    let data = NSMutableData()
    guard let destination = CGImageDestinationCreateWithData(data, UTType.png.identifier as CFString, 1, nil) else { return nil }
    CGImageDestinationAddImage(destination, image, nil)
    guard CGImageDestinationFinalize(destination) else { return nil }
    return data as Data
}

func screenshot() -> (Data, Int, Int)? {
    guard let image = CGWindowListCreateImage(CGRect.infinite, .optionOnScreenOnly, kCGNullWindowID, [.bestResolution, .boundsIgnoreFraming]), let data = pngData(image) else { return nil }
    return (data, image.width, image.height)
}

func base64(_ data: Data) -> String { data.base64EncodedString() }

func keyCode(_ key: String) -> CGKeyCode? {
    let map: [String: CGKeyCode] = ["return": 36, "enter": 36, "tab": 48, "space": 49, "delete": 51, "escape": 53, "esc": 53, "left": 123, "right": 124, "down": 125, "up": 126, "a": 0, "b": 11, "c": 8, "d": 2, "e": 14, "f": 3, "g": 5, "h": 4, "i": 34, "j": 38, "k": 40, "l": 37, "m": 46, "n": 45, "o": 31, "p": 35, "q": 12, "r": 15, "s": 1, "t": 17, "u": 32, "v": 9, "w": 13, "x": 7, "y": 16, "z": 6]
    return map[key.lowercased()]
}

func postKey(_ code: CGKeyCode, down: Bool, flags: CGEventFlags = []) {
    guard let event = CGEvent(keyboardEventSource: nil, virtualKey: code, keyDown: down) else { return }
    event.flags = flags
    event.post(tap: .cghidEventTap)
}

func execute(_ payload: JSONValue) -> (Bool, String?, String?) {
    guard case .object(let object) = payload, case .string(let kind) = object["kind"] else { return (false, "invalid_action", "action kind is required") }
    if stopped { return (false, "stopped", "helper is stopped") }
    if paused { return (false, "paused", "helper is paused") }
    switch kind {
    case "click", "double_click", "right_click", "move":
        guard let x = object["x"].flatMap({ if case .number(let value) = $0 { return Int(value) }; return nil }), let y = object["y"].flatMap({ if case .number(let value) = $0 { return Int(value) }; return nil }) else { return (false, "invalid_action", "point is required") }
        let point = CGPoint(x: x, y: y)
        let button: CGMouseButton = kind == "right_click" ? .right : .left
        let down = kind == "move" ? CGEventType.mouseMoved : (button == .right ? .rightMouseDown : .leftMouseDown)
        let up = kind == "move" ? CGEventType.mouseMoved : (button == .right ? .rightMouseUp : .leftMouseUp)
        CGEvent(mouseEventSource: nil, mouseType: down, mouseCursorPosition: point, mouseButton: button)?.post(tap: .cghidEventTap)
        if kind != "move" {
            if kind == "double_click" { usleep(60000) }
            CGEvent(mouseEventSource: nil, mouseType: up, mouseCursorPosition: point, mouseButton: button)?.post(tap: .cghidEventTap)
            if kind == "double_click" {
                usleep(60000)
                CGEvent(mouseEventSource: nil, mouseType: down, mouseCursorPosition: point, mouseButton: button)?.post(tap: .cghidEventTap)
                CGEvent(mouseEventSource: nil, mouseType: up, mouseCursorPosition: point, mouseButton: button)?.post(tap: .cghidEventTap)
            }
        }
        return (true, nil, nil)
    case "type":
        guard case .string(let text) = object["text"] else { return (false, "invalid_action", "text is required") }
        guard let event = CGEvent(keyboardEventSource: nil, virtualKey: 0, keyDown: true) else { return (false, "input_unavailable", "keyboard event unavailable") }
        event.keyboardSetUnicodeString(stringLength: text.utf16.count, unicodeString: Array(text.utf16))
        event.post(tap: .cghidEventTap)
        return (true, nil, nil)
    case "key":
        guard case .string(let key) = object["key"], let code = keyCode(key) else { return (false, "invalid_key", "unsupported key") }
        postKey(code, down: true); postKey(code, down: false)
        return (true, nil, nil)
    case "hotkey":
        guard case .array(let values) = object["keys"] else { return (false, "invalid_action", "keys are required") }
        var codes: [CGKeyCode] = []
        for value in values { if case .string(let key) = value, let code = keyCode(key) { codes.append(code) } }
        guard !codes.isEmpty else { return (false, "invalid_key", "unsupported hotkey") }
        for code in codes { postKey(code, down: true) }
        for code in codes.reversed() { postKey(code, down: false) }
        return (true, nil, nil)
    case "scroll":
        let dy = object["delta_y"].flatMap({ if case .number(let value) = $0 { return Int(value) }; return nil }) ?? 0
        let dx = object["delta_x"].flatMap({ if case .number(let value) = $0 { return Int(value) }; return nil }) ?? 0
        CGEvent(scrollWheelEvent2Source: nil, units: .pixel, wheelCount: 2, wheel1: Int32(dy), wheel2: Int32(dx), wheel3: 0)?.post(tap: .cghidEventTap)
        return (true, nil, nil)
    case "wait":
        let ms = object["duration_ms"].flatMap({ if case .number(let value) = $0 { return Int(value) }; return nil }) ?? 0
        usleep(UInt32(max(0, ms)) * 1000)
        return (true, nil, nil)
    default:
        return (false, "unsupported_action", "unsupported action")
    }
}

while let data = readFrame() {
    guard let request = try? JSONDecoder().decode(Envelope.self, from: data) else { continue }
    guard request.protocolVersion == protocolVersion else { response(for: request, ok: false, outcome: "rejected", errorCode: "protocol_mismatch", errorMessage: "unsupported protocol version"); continue }
    switch request.command {
    case "readiness": response(for: request, ok: true, outcome: "executed", payload: readiness())
    case "observe":
        guard CGPreflightScreenCaptureAccess() else { response(for: request, ok: false, outcome: "rejected", errorCode: "screen_recording_required", errorMessage: "Screen Recording permission is required"); continue }
        guard let (data, width, height) = screenshot() else { response(for: request, ok: false, outcome: "failed", errorCode: "screenshot_failed", errorMessage: "unable to capture desktop screenshot"); continue }
        response(for: request, ok: true, outcome: "executed", payload: .object(["media_type": .string("image/png"), "data": .string(base64(data)), "width": .number(Double(width)), "height": .number(Double(height))]))
    case "execute":
        guard let payload = request.payload else { response(for: request, ok: false, outcome: "rejected", errorCode: "invalid_action", errorMessage: "action payload is required"); continue }
        let result = execute(payload)
        if !result.0 { response(for: request, ok: false, outcome: "failed", errorCode: result.1, errorMessage: result.2); continue }
        guard let (data, width, height) = screenshot() else { response(for: request, ok: true, outcome: "unknown", errorCode: "after_screenshot_failed", errorMessage: "input may have been delivered but after screenshot failed"); continue }
        response(for: request, ok: true, outcome: "executed", payload: .object(["media_type": .string("image/png"), "data": .string(base64(data)), "width": .number(Double(width)), "height": .number(Double(height))]))
    case "pause": paused = true; response(for: request, ok: true, outcome: "executed")
    case "resume": if stopped { response(for: request, ok: false, outcome: "rejected", errorCode: "stopped", errorMessage: "helper is stopped") } else { paused = false; response(for: request, ok: true, outcome: "executed") }
    case "stop": stopped = true; paused = true; response(for: request, ok: true, outcome: "executed")
    case "shutdown": response(for: request, ok: true, outcome: "executed"); exit(0)
    default: response(for: request, ok: false, outcome: "rejected", errorCode: "unsupported_command", errorMessage: "unsupported command")
    }
}
