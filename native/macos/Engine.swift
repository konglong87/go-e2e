import Foundation
import CoreGraphics

struct ActionResult {
    let outcome: Outcome
    let payload: JSONValue
    let error: SafetyError?
    var dispatchState: DispatchState? = nil
}

private struct CapturedSnapshot {
    let payload: JSONValue
    let geometry: DisplayGeometry
    let window: NativeWindow?
    let focus: Int32
}

typealias NativeDiagnosticWriter = ([String: Any]) -> Void

final class Engine {
    let state: SafetyState
    private let platform: DesktopPlatform
    private let now: () -> Date
    private let diagnostic: NativeDiagnosticWriter

    init(platform: DesktopPlatform, now: @escaping () -> Date = Date.init,
         diagnostic: @escaping NativeDiagnosticWriter = { _ in }) {
        self.platform = platform; self.now = now; self.diagnostic = diagnostic; self.state = SafetyState(now: now)
    }

    private func record(_ request: Envelope, phase: String, error: SafetyError? = nil,
                        fields: [String: Any] = [:]) {
        var event: [String: Any] = [
            "layer": "swift_helper", "phase": phase, "command": request.command,
            "request_id": request.requestID, "action_id": request.actionID, "session_id": request.sessionID,
        ]
        if let error { event["error_code"] = error.rawValue }
        if let targetID = request.payload["target_id"]?.string { event["target_id"] = targetID }
        if let windowID = request.payload["window_id"]?.string { event["requested_window_id"] = windowID }
        if let displayID = request.payload["display_id"]?.string { event["requested_display_id"] = displayID }
        for (key, value) in fields { event[key] = value }
        diagnostic(event)
    }

    private func recordWindowMismatch(_ request: Envelope, phase: String, reason: String,
                                      expected: NativeWindow? = nil, actual: NativeWindow? = nil) {
        var fields: [String: Any] = ["reason": reason]
        if let expected { fields["expected_window"] = expected.diagnosticFields }
        if let actual { fields["actual_window"] = actual.diagnosticFields }
        if let expected, let actual { fields["identity_diffs"] = expected.identityDiffs(actual) }
        record(request, phase: phase, error: .targetWindowMismatch, fields: fields)
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
            guard window.bundleID == target.bundleID else {
                recordWindowMismatch(request, phase: "launch_bundle_validation", reason: "bundle_id_mismatch", actual: window)
                throw SafetyError.targetWindowMismatch
            }
            try state.bindTarget(window, expectedBundleID: target.bundleID, for: request)
            record(request, phase: "launch_result", fields: ["result": "success", "window": window.diagnosticFields])
            return ActionResult(outcome: .executed, payload: .object([
                "operation": .string(Command.launchApp.rawValue),
                "target_id": .string(target.id),
                "display_name": .string(target.displayName),
                "bundle_id": .string(target.bundleID),
                "window": window.json
            ]), error: nil)
        } catch {
            let safetyError = error as? SafetyError ?? .launchFailed
            record(request, phase: "launch_result", error: safetyError, fields: ["result": "error"])
            return ActionResult(outcome: .rejected, payload: .object([:]), error: safetyError)
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
        guard request.payload["window_id"]?.string == bound.id else {
            recordWindowMismatch(request, phase: "authorized_window", reason: "requested_window_id_mismatch", expected: bound)
            throw SafetyError.targetWindowMismatch
        }
        do {
            guard let current = try requestedWindow(request) else {
                recordWindowMismatch(request, phase: "authorized_window", reason: "requested_window_not_found", expected: bound)
                throw SafetyError.targetWindowMismatch
            }
            let differences = bound.identityDiffs(current)
            guard differences.isEmpty else {
                recordWindowMismatch(request, phase: "authorized_window", reason: "identity_mismatch", expected: bound, actual: current)
                throw SafetyError.targetWindowMismatch
            }
            return current
        } catch let error as SafetyError {
            if error == .unsupportedDisplay {
                recordWindowMismatch(request, phase: "authorized_window", reason: "window_unavailable", expected: bound)
            }
            throw error == .unsupportedDisplay ? SafetyError.targetWindowMismatch : error
        } catch {
            recordWindowMismatch(request, phase: "authorized_window", reason: "window_inventory_error", expected: bound)
            throw SafetyError.targetWindowMismatch
        }
    }

    private func target(_ request: Envelope, geometry: DisplayGeometry) throws {
        if let bound = try state.target(for: request) {
            guard geometry.windowID == bound.id else {
                recordWindowMismatch(request, phase: "target_validation", reason: "geometry_window_id_mismatch", expected: bound,
                                     actual: try? requestedWindow(request))
                throw SafetyError.targetWindowMismatch
            }
            guard try authorizedWindow(request) != nil else { throw SafetyError.targetWindowMismatch }
        } else if let window = request.payload["window_id"]?.string, !window.isEmpty && geometry.windowID != window {
            record(request, phase: "target_validation", error: .unsupportedDisplay,
                   fields: ["reason": "requested_window_geometry_mismatch", "geometry_window_id": geometry.windowID ?? ""])
            throw SafetyError.unsupportedDisplay
        }
        if let display = request.payload["display_id"]?.string, !display.isEmpty && display != geometry.id {
            record(request, phase: "target_validation", error: .unsupportedDisplay,
                   fields: ["reason": "display_id_mismatch", "geometry_display_id": geometry.id])
            throw SafetyError.unsupportedDisplay
        }
    }
    private func displayIDs() throws -> [String] {
        try platform.geometries().map(\.id).sorted()
    }
    // Pure revalidation: never raise/activate a window while checking input or
    // capturing evidence. A focus change must remain visible to the caller.
    private func checkWindow(_ request: Envelope, geometry: DisplayGeometry, expectedWindow: NativeWindow?) throws {
        if let expectedWindow {
            guard let current = try requestedWindow(request) else {
                recordWindowMismatch(request, phase: "check_window", reason: "requested_window_not_found", expected: expectedWindow)
                throw SafetyError.targetWindowMismatch
            }
            let differences = expectedWindow.identityDiffs(current)
            guard differences.isEmpty else {
                recordWindowMismatch(request, phase: "check_window", reason: "identity_mismatch", expected: expectedWindow, actual: current)
                throw SafetyError.targetWindowMismatch
            }
        }
        if let windowID = geometry.windowID, platform.activeWindowID() != windowID {
            record(request, phase: "check_window", error: .focusChanged,
                   fields: ["reason": "active_window_mismatch", "geometry_window_id": windowID,
                            "active_window_id": platform.activeWindowID() ?? ""])
            throw SafetyError.focusChanged
        }
    }
    private func capture(_ request: Envelope, geometry: DisplayGeometry, expectedWindow: NativeWindow? = nil) throws -> CapturedSnapshot {
        try state.gate(request)
        let capOK = platform.captureAllowed()
        // The window position (bounds.origin) can shift between observe and
        // capture by a few pixels during launch animation or Dock placement.
        // Identity (display id, window id, pixel dimensions) must remain
        // stable; use the freshly resolved geometry so the screenshot reflects
        // the window's real position, and reject only if identity changed.
        let resolved = try resolveGeometry(for: request)
        guard capOK,
              resolved.id == geometry.id,
              resolved.windowID == geometry.windowID,
              resolved.width == geometry.width,
              resolved.height == geometry.height else {
            record(request, phase: "capture_precheck", error: .screenshotFailed,
                   fields: ["reason": "geometry_changed", "expected_display_id": geometry.id,
                            "actual_display_id": resolved.id, "expected_window_id": geometry.windowID ?? "",
                            "actual_window_id": resolved.windowID ?? "", "expected_width": geometry.width,
                            "expected_height": geometry.height, "actual_width": resolved.width,
                            "actual_height": resolved.height])
            throw SafetyError.screenshotFailed
        }
        let active = resolved
        try checkWindow(request, geometry: active, expectedWindow: expectedWindow)
        guard let focus = platform.focus() else { throw SafetyError.focusChanged }
        let data = try platform.capture(active)
        try state.gate(request)
        let postCheck = try resolveGeometry(for: request)
        guard postCheck.id == active.id, postCheck.windowID == active.windowID,
              postCheck.width == active.width, postCheck.height == active.height, postCheck.bounds == active.bounds else {
            record(request, phase: "capture_postcheck", error: .unsupportedDisplay,
                   fields: ["reason": "geometry_changed_after_capture", "window_id": active.windowID ?? "",
                            "post_window_id": postCheck.windowID ?? "", "width": active.width,
                            "height": active.height, "post_width": postCheck.width, "post_height": postCheck.height])
            throw SafetyError.unsupportedDisplay
        }
        try checkWindow(request, geometry: active, expectedWindow: expectedWindow)
        guard platform.focus() == focus else { throw SafetyError.focusChanged }
        var result: [String: JSONValue] = ["media_type": .string("image/png"), "data": .string(data.base64EncodedString()),
                        "width": .number(Double(active.width)), "height": .number(Double(active.height)),
                        "scale_factor": .number(active.scale), "display_id": .string(active.id),
                        "coordinate_space": active.coordinateSpace]
        let target = try requestedWindow(request)
        if let windowID = active.windowID {
            result["window_id"] = .string(windowID)
            guard let target, target.frame == active.bounds else { throw SafetyError.screenshotFailed }
            if let expectedWindow, !target.matchesIdentity(expectedWindow) { throw SafetyError.targetWindowMismatch }
            result["target_window"] = target.json
        }
        if let frontmost = try? platform.windows().first(where: { $0.isFrontmost }) { result["active_window"] = frontmost.json }
        return CapturedSnapshot(payload: .object(result), geometry: active, window: target, focus: focus)
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

    private func captureObservation(_ request: Envelope, expectedWindow: NativeWindow?) throws -> CapturedSnapshot {
        for attempt in 0..<3 {
            do {
                let geometry = try resolveGeometry(for: request)
                try target(request, geometry: geometry)
                return try capture(request, geometry: geometry, expectedWindow: expectedWindow)
            } catch let error as SafetyError where error == .screenshotFailed && attempt < 2 {
                try wait(50 * (attempt + 1), request: request)
            }
        }
        throw SafetyError.screenshotFailed
    }

    func observe(_ request: Envelope) -> ActionResult {
        do {
            try state.gate(request)
            let window = try authorizedWindow(request)
            if let window { try activateTarget(window, request: request) }
            guard let id = request.payload["observation_id"]?.string, !id.isEmpty, id.utf8.count <= 256,
                  let focus = platform.focus() else { throw SafetyError.invalidAction }
            let captured = try captureObservation(request, expectedWindow: window)
            guard case .object(var payload) = captured.payload else { throw SafetyError.screenshotFailed }
            guard platform.focus() == focus else { throw SafetyError.focusChanged }
            if let windowID = captured.geometry.windowID, platform.activeWindowID() != windowID { throw SafetyError.focusChanged }
            let expires = now().addingTimeInterval(observationTTLSeconds)
            let topology = try displayIDs()
            try state.save(Snapshot(id: id, session: request.sessionID, geometry: captured.geometry, window: captured.window, displayIDs: topology, focus: captured.focus, expires: expires), for: request)
            let formatter = ISO8601DateFormatter()
            formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            payload["observation_expires_at"] = .string(formatter.string(from: expires))
            return ActionResult(outcome: .executed, payload: .object(payload), error: nil)
        } catch {
            let safetyError = error as? SafetyError ?? .screenshotFailed
            record(request, phase: "observe_result", error: safetyError, fields: ["result": "error"])
            return ActionResult(outcome: .rejected, payload: .object([:]), error: safetyError)
        }
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
                return try capture(request, geometry: geometry, expectedWindow: expectedWindow).payload
            } catch let error as SafetyError where error == .screenshotFailed && attempt < 2 {
                Thread.sleep(forTimeInterval: 0.05 * Double(attempt + 1))
            }
        }
        return try capture(request, geometry: geometry, expectedWindow: expectedWindow).payload
    }
    func execute(_ request: Envelope) -> ActionResult {
        var posted = false
        var inputComplete = false
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
            inputComplete = !plan.operations.isEmpty
            try wait(plan.waitMS, request: request)
            // The final atomic operation may legitimately activate another app.
            // Capture requires stable current focus, not the consumed binding;
            // subsequent input still needs a fresh observe. This is not visual
            // verification that the action achieved its intended result.
            let payload = try captureAfterInput(request, geometry: snapshot.geometry, expectedWindow: snapshot.window)
            return ActionResult(outcome: .executed, payload: payload, error: nil,
                                dispatchState: inputComplete ? .complete : .notStarted)
        } catch {
            // Explicit Pause/Stop already invalidated the generation. Preserve
            // that state so Pause can Resume with a fresh observation; do not
            // turn cooperative interruption into an irreversible helper stop.
            if (error as? SafetyError) == .inputUncertain { posted = true }
            let safetyError = error as? SafetyError ?? .inputUnavailable
            record(request, phase: "execute_result", error: safetyError,
                   fields: ["result": inputComplete ? "executed" : (posted ? "unknown" : "rejected"),
                            "dispatch_state": inputComplete ? "complete" : (posted ? "unknown" : "not_started"), "input_posted": posted])
            if inputComplete && safetyError != .inputUncertain {
                // All planned pairs were acknowledged. Losing only evidence
                // must not turn confirmed dispatch into partial/unknown input.
                return ActionResult(outcome: .executed, payload: .object([:]), error: safetyError, dispatchState: .complete)
            }
            if posted && safetyError != .inactive { state.end() }
            return ActionResult(outcome: posted ? .unknown : .rejected, payload: .object([:]), error: safetyError,
                                dispatchState: posted ? .unknown : .notStarted)
        }
    }
}
