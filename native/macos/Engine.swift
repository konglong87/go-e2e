import Foundation
import CoreGraphics

struct ActionResult {
    let outcome: Outcome
    let payload: JSONValue
    let error: SafetyError?
}
final class Engine {
    let state: SafetyState
    private let platform: DesktopPlatform
    private let now: () -> Date
    init(platform: DesktopPlatform, now: @escaping () -> Date = Date.init) {
        self.platform = platform; self.now = now; self.state = SafetyState(now: now)
    }

    func requestPermissions() -> JSONValue {
        platform.requestPermissions()
        return readiness()
    }

    func readiness() -> JSONValue {
        let geometries = try? platform.geometries()
        let primary = try? platform.geometry()
        let geometry = primary ?? geometries?.first
        let capture = platform.captureAllowed(), input = platform.inputAllowed()
        let displays = geometries?.map { $0.coordinateSpace } ?? []
        let windows = (try? platform.windows())?.map { $0.json } ?? []
        return .object([
            "capture_readiness": .string(geometry == nil ? "unavailable" : (capture ? "ready" : "permission_required")),
            "input_readiness": .string(geometry == nil ? "unavailable" : (input ? "ready" : "permission_required")),
            "permission_state": .string(capture && input ? "approved" : "required"),
            "focus_state": .string(platform.focus() == nil ? "unavailable" : "focused"),
            "image_supported": .bool(geometry != nil), "supports_pause": .bool(true), "supports_stop": .bool(true),
            "coordinate_space": geometry?.coordinateSpace ?? .object(["origin":.string("top_left"),"unit":.string("pixels"),"width":.number(0),"height":.number(0),"scale_factor":.number(0)]),
            "displays": .array(displays),
            "windows": .array(windows),
            "actions": .array(ActionKind.allCases.filter { $0 != .drag || platform.supportsDrag() }.map { .string($0.rawValue) })
        ])
    }
    private func requestedTarget(_ payload: JSONValue) throws -> (id: String, displayName: String, bundleID: String) {
        guard case .object(let fields) = payload,
              let id = fields["target_id"]?.string, !id.isEmpty,
              let displayName = fields["display_name"]?.string, !displayName.isEmpty,
              let bundleID = fields["bundle_id"]?.string, !bundleID.isEmpty,
              fields.keys.allSatisfy({ $0 == "target_id" || $0 == "display_name" || $0 == "bundle_id" || $0 == "generation" }) else {
            throw SafetyError.unsupportedApplication
        }
        // The signed host resolves target IDs to an allowlisted bundle before
        // reaching the helper. The helper still rejects malformed bundle IDs
        // and binds the discovered window to the exact requested identity.
        guard bundleID.split(separator: ".").count >= 2,
              bundleID.utf8.count <= 256 else { throw SafetyError.unsupportedApplication }
        return (id, displayName, bundleID)
    }
    func launchApp(_ request: Envelope) -> ActionResult {
        do {
            let target = try requestedTarget(request.payload)
            try state.beginControl(request)
            let window = try platform.launchApplication(bundleID: target.bundleID) { self.state.permitted(request) }
            guard window.bundleID == target.bundleID else { throw SafetyError.targetWindowMismatch }
            try state.bindTarget(window, expectedBundleID: target.bundleID, for: request)
            return ActionResult(outcome: .executed, payload: .object([
                "operation": .string(Command.launchApp.rawValue),
                "target_id": .string(target.id),
                "display_name": .string(target.displayName),
                "bundle_id": .string(target.bundleID),
                "window": window.json
            ]), error: nil)
        } catch {
            return ActionResult(outcome: .rejected, payload: .object([:]), error: error as? SafetyError ?? .launchFailed)
        }
    }
    private func resolveGeometry(for request: Envelope) throws -> DisplayGeometry {
        if let windowID = request.payload["window_id"]?.string, !windowID.isEmpty {
            return try platform.windowGeometry(windowID)
        }
        let all = try platform.geometries()
        if let display = request.payload["display_id"]?.string, !display.isEmpty {
            guard let selected = all.first(where: { $0.id == display }) else { throw SafetyError.unsupportedDisplay }
            return selected
        }
        return try platform.geometry()
    }
    private func requestedWindow(_ request: Envelope) throws -> NativeWindow? {
        guard let id = request.payload["window_id"]?.string, !id.isEmpty else { return nil }
        guard let window = try platform.windows().first(where: { $0.id == id }) else { throw SafetyError.unsupportedDisplay }
        return window
    }
    private func authorizedWindow(_ request: Envelope) throws -> NativeWindow? {
        guard let bound = try state.target(for: request) else { return try requestedWindow(request) }
        guard request.payload["window_id"]?.string == bound.id else { throw SafetyError.targetWindowMismatch }
        do {
            guard let current = try requestedWindow(request), current.matchesIdentity(bound) else {
                throw SafetyError.targetWindowMismatch
            }
            return bound
        } catch let error as SafetyError {
            throw error == .unsupportedDisplay ? SafetyError.targetWindowMismatch : error
        } catch {
            throw SafetyError.targetWindowMismatch
        }
    }

    private func target(_ request: Envelope, geometry: DisplayGeometry) throws {
        if let bound = try state.target(for: request) {
            guard geometry.windowID == bound.id, try authorizedWindow(request) != nil else {
                throw SafetyError.targetWindowMismatch
            }
        } else if let window = request.payload["window_id"]?.string, !window.isEmpty && geometry.windowID != window {
            throw SafetyError.unsupportedDisplay
        }
        if let display = request.payload["display_id"]?.string, !display.isEmpty && display != geometry.id { throw SafetyError.unsupportedDisplay }
    }
    private func displayIDs() throws -> [String] {
        try platform.geometries().map(\.id).sorted()
    }
    // Pure revalidation: never raise/activate a window while checking input or
    // capturing evidence. A focus change must remain visible to the caller.
    private func checkWindow(_ request: Envelope, geometry: DisplayGeometry, expectedWindow: NativeWindow?) throws {
        if let expectedWindow {
            guard let current = try requestedWindow(request), current.matchesIdentity(expectedWindow) else { throw SafetyError.targetWindowMismatch }
        }
        if let windowID = geometry.windowID, platform.activeWindowID() != windowID { throw SafetyError.focusChanged }
    }
    private func capture(_ request: Envelope, geometry: DisplayGeometry, expectedWindow: NativeWindow? = nil) throws -> JSONValue {
        try state.gate(request)
        guard platform.captureAllowed(), try resolveGeometry(for: request) == geometry else { throw SafetyError.screenshotFailed }
        try checkWindow(request, geometry: geometry, expectedWindow: expectedWindow)
        guard let focus = platform.focus() else { throw SafetyError.focusChanged }
        let data = try platform.capture(geometry)
        try state.gate(request)
        guard try resolveGeometry(for: request) == geometry else { throw SafetyError.unsupportedDisplay }
        try checkWindow(request, geometry: geometry, expectedWindow: expectedWindow)
        guard platform.focus() == focus else { throw SafetyError.focusChanged }
        var result: [String: JSONValue] = ["media_type": .string("image/png"), "data": .string(data.base64EncodedString()),
                        "width": .number(Double(geometry.width)), "height": .number(Double(geometry.height)),
                        "scale_factor": .number(geometry.scale), "display_id": .string(geometry.id),
                        "coordinate_space": geometry.coordinateSpace]
        if let windowID = geometry.windowID {
            result["window_id"] = .string(windowID)
            guard let target = try requestedWindow(request) else { throw SafetyError.unsupportedDisplay }
            result["target_window"] = target.json
        }
        if let active = try? platform.windows().first(where: { $0.isFrontmost }) { result["active_window"] = active.json }
        return .object(result)
    }
    private static let activationTimeout = 1.0
    private static let activationPollMS = 20

    private func activateTarget(_ window: NativeWindow, request: Envelope) throws {
        guard platform.captureAllowed(), platform.inputAllowed() else { throw SafetyError.permissionRequired }
        guard window.isVisible else { throw SafetyError.unsupportedDisplay }
        let activated = platform.activateWindow(window.id, permitted: {
            do { try self.state.gate(request); return true } catch { return false }
        })
        try state.gate(request)
        guard activated else { throw SafetyError.focusChanged }
        // Activation/AXRaise acknowledgements are asynchronous. Confirm the
        // exact Window Server target, not merely the application's PID.
        let until = ProcessInfo.processInfo.systemUptime + Self.activationTimeout
        while true {
            try state.gate(request)
            guard platform.captureAllowed(), platform.inputAllowed() else { throw SafetyError.permissionRequired }
            if platform.focus() == window.ownerPID && platform.activeWindowID() == window.id { return }
            guard ProcessInfo.processInfo.systemUptime < until else { throw SafetyError.focusChanged }
            try wait(Self.activationPollMS, request: request)
        }
    }

    func observe(_ request: Envelope) -> ActionResult {
        do {
            try state.gate(request)
            let window = try authorizedWindow(request)
            if let window { try activateTarget(window, request: request) }
            let geometry = try resolveGeometry(for: request); try target(request, geometry: geometry)
            guard let id = request.payload["observation_id"]?.string, !id.isEmpty, id.utf8.count <= 256,
                  let focus = platform.focus() else { throw SafetyError.invalidAction }
            guard case .object(var payload) = try capture(request, geometry: geometry, expectedWindow: window) else { throw SafetyError.screenshotFailed }
            guard platform.focus() == focus else { throw SafetyError.focusChanged }
            if let windowID = geometry.windowID, platform.activeWindowID() != windowID { throw SafetyError.focusChanged }
            let expires = now().addingTimeInterval(observationTTLSeconds)
            let topology = try displayIDs()
            try state.save(Snapshot(id: id, session: request.sessionID, geometry: geometry, window: window, displayIDs: topology, focus: focus, expires: expires), for: request)
            let formatter = ISO8601DateFormatter()
            formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            payload["observation_expires_at"] = .string(formatter.string(from: expires))
            return ActionResult(outcome: .executed, payload: .object(payload), error: nil)
        } catch { return ActionResult(outcome: .rejected, payload: .object([:]), error: error as? SafetyError ?? .screenshotFailed) }
    }
    private func checkInput(_ snapshot: Snapshot, request: Envelope) throws {
        guard platform.inputAllowed(), platform.captureAllowed() else { throw SafetyError.permissionRequired }
        guard platform.focus() == snapshot.focus else { throw SafetyError.focusChanged }
        try checkWindow(request, geometry: snapshot.geometry, expectedWindow: snapshot.window)
        guard try resolveGeometry(for: request) == snapshot.geometry else { throw SafetyError.unsupportedDisplay }
        guard try displayIDs() == snapshot.displayIDs else { throw SafetyError.unsupportedDisplay }
        // Native discovery can take time. Recheck focus after it, immediately
        // before entering the short event-post gate.
        guard platform.focus() == snapshot.focus else { throw SafetyError.focusChanged }
        try checkWindow(request, geometry: snapshot.geometry, expectedWindow: snapshot.window)
    }
    private func wait(_ milliseconds: Int, request: Envelope) throws {
        let end = ProcessInfo.processInfo.systemUptime + Double(milliseconds) / 1000
        while ProcessInfo.processInfo.systemUptime < end {
            try state.gate(request)
            Thread.sleep(forTimeInterval: min(0.01, max(0, end - ProcessInfo.processInfo.systemUptime)))
        }
        try state.gate(request)
    }
    // Launching an app from the Dock can take longer than a normal input
    // screenshot. Wait for a frontmost-app transition when the click lands in
    // the Dock band, then retry only the screenshot (never the input). This
    // keeps the returned evidence aligned with the actual post-click state
    // without weakening focus, permission, geometry, or topology guards.
    private func waitsForAppActivation(_ request: Envelope, geometry: DisplayGeometry) -> Bool {
        guard let kind = request.payload["kind"]?.string,
              ["click", "double_click", "right_click"].contains(kind),
              let y = request.payload["y"]?.integer(in: 0...maxImageDimension) else { return false }
        return y >= max(0, geometry.height - 220)
    }

    private func waitForAppActivation(_ request: Envelope, geometry: DisplayGeometry) throws {
        guard waitsForAppActivation(request, geometry: geometry), let before = platform.focus() else { return }
        let deadline = ProcessInfo.processInfo.systemUptime + 3.0
        while ProcessInfo.processInfo.systemUptime < deadline {
            try state.gate(request)
            if platform.focus() != before { return }
            try wait(100, request: request)
        }
    }

    private func captureAfterInput(_ request: Envelope, geometry: DisplayGeometry, expectedWindow: NativeWindow?) throws -> JSONValue {
        try waitForAppActivation(request, geometry: geometry)
        for attempt in 0..<3 {
            do {
                return try capture(request, geometry: geometry, expectedWindow: expectedWindow)
            } catch let error as SafetyError where error == .screenshotFailed && attempt < 2 {
                Thread.sleep(forTimeInterval: 0.05 * Double(attempt + 1))
            }
        }
        return try capture(request, geometry: geometry, expectedWindow: expectedWindow)
    }
    func execute(_ request: Envelope) -> ActionResult {
        var posted = false
        var pressedButton: CGMouseButton?
        defer { if let pressedButton { platform.releasePressedButton(pressedButton) } }
        do {
            let snapshot = try state.begin(request); try target(request, geometry: snapshot.geometry)
            let plan = try ActionPlan(request.payload, geometry: snapshot.geometry)
            // Passive wait posts no input: a legitimate app launch can change
            // focus after the preceding screenshot. Keep session/generation/
            // expiry binding above and stable-target capture below, but reserve
            // stale-input preflight for operations that actually post events.
            if !plan.operations.isEmpty {
                try checkInput(snapshot, request: request)
                try platform.prepareInput(request)
            }
            for (index, operation) in plan.operations.enumerated() {
                if plan.doubleClick && index > 0 { try wait(60, request: request) }
                // Revalidate TCC/focus/display just before each pair. Stop only
                // waits for this short critical section, not the entire action.
                try checkInput(snapshot, request: request)
                if case .mouseDown(_, let button) = operation { pressedButton = button }
                try state.gate(request) {
                    try platform.post(operation); posted = true
                }
                if case .mouseUp = operation { pressedButton = nil }
                if plan.dragStepDelayMS > 0 && index + 1 < plan.operations.count {
                    try wait(plan.dragStepDelayMS, request: request)
                }
            }
            try wait(plan.waitMS, request: request)
            // The final atomic operation may legitimately activate another app.
            // Capture requires stable current focus, not the consumed binding;
            // subsequent input still needs a fresh observe. This is not visual
            // verification that the action achieved its intended result.
            let payload = try captureAfterInput(request, geometry: snapshot.geometry, expectedWindow: snapshot.window)
            return ActionResult(outcome: .executed, payload: payload, error: nil)
        } catch {
            // Explicit Pause/Stop already invalidated the generation. Preserve
            // that state so Pause can Resume with a fresh observation; do not
            // turn cooperative interruption into an irreversible helper stop.
            if (error as? SafetyError) == .inputUncertain { posted = true }
            if posted && (error as? SafetyError) != .inactive { state.end() }
            return ActionResult(outcome: posted ? .unknown : .rejected, payload: .object([:]), error: error as? SafetyError ?? .inputUnavailable)
        }
    }
}
