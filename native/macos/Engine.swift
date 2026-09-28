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
            "actions": .array(ActionKind.allCases.map { .string($0.rawValue) })
        ])
    }
    private func resolveGeometry(for request: Envelope) throws -> DisplayGeometry {
        if let windowID = request.payload["window_id"]?.string, !windowID.isEmpty {
            guard platform.activateWindow(windowID) else { throw SafetyError.focusChanged }
            return try platform.windowGeometry(windowID)
        }
        let all = try platform.geometries()
        if let display = request.payload["display_id"]?.string, !display.isEmpty {
            guard let selected = all.first(where: { $0.id == display }) else { throw SafetyError.unsupportedDisplay }
            return selected
        }
        return try platform.geometry()
    }
    private func target(_ request: Envelope, geometry: DisplayGeometry) throws {
        if let window = request.payload["window_id"]?.string, !window.isEmpty && geometry.windowID != window { throw SafetyError.unsupportedDisplay }
        if let display = request.payload["display_id"]?.string, !display.isEmpty && display != geometry.id { throw SafetyError.unsupportedDisplay }
    }
    private func displayIDs() throws -> [String] {
        try platform.geometries().map(\.id).sorted()
    }
    private func capture(_ request: Envelope, geometry: DisplayGeometry) throws -> JSONValue {
        try state.gate(request)
        guard platform.captureAllowed(), try resolveGeometry(for: request) == geometry else { throw SafetyError.screenshotFailed }
        guard let focus = platform.focus() else { throw SafetyError.focusChanged }
        let data = try platform.capture(geometry)
        try state.gate(request)
        guard try resolveGeometry(for: request) == geometry else { throw SafetyError.unsupportedDisplay }
        guard platform.focus() == focus else { throw SafetyError.focusChanged }
        if let windowID = geometry.windowID, platform.activeWindowID() != windowID { throw SafetyError.focusChanged }
        var result: [String: JSONValue] = ["media_type": .string("image/png"), "data": .string(data.base64EncodedString()),
                        "width": .number(Double(geometry.width)), "height": .number(Double(geometry.height)),
                        "scale_factor": .number(geometry.scale), "display_id": .string(geometry.id)]
        if let windowID = geometry.windowID { result["window_id"] = .string(windowID) }
        if let active = try? platform.windows().first(where: { $0.isFrontmost }) { result["active_window"] = active.json }
        return .object(result)
    }
    func observe(_ request: Envelope) -> ActionResult {
        do {
            try state.gate(request)
            let geometry = try resolveGeometry(for: request); try target(request, geometry: geometry)
            guard let id = request.payload["observation_id"]?.string, !id.isEmpty, id.utf8.count <= 256,
                  let focus = platform.focus() else { throw SafetyError.invalidAction }
            guard case .object(var payload) = try capture(request, geometry: geometry) else { throw SafetyError.screenshotFailed }
            guard platform.focus() == focus else { throw SafetyError.focusChanged }
            if let windowID = geometry.windowID, platform.activeWindowID() != windowID { throw SafetyError.focusChanged }
            let expires = now().addingTimeInterval(observationTTLSeconds)
            let topology = try displayIDs()
            try state.save(Snapshot(id: id, session: request.sessionID, geometry: geometry, displayIDs: topology, focus: focus, expires: expires), for: request)
            let formatter = ISO8601DateFormatter()
            formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
            payload["observation_expires_at"] = .string(formatter.string(from: expires))
            return ActionResult(outcome: .executed, payload: .object(payload), error: nil)
        } catch { return ActionResult(outcome: .rejected, payload: .object([:]), error: error as? SafetyError ?? .screenshotFailed) }
    }
    private func checkInput(_ snapshot: Snapshot, request: Envelope) throws {
        guard platform.inputAllowed(), platform.captureAllowed() else { throw SafetyError.permissionRequired }
        guard platform.focus() == snapshot.focus else { throw SafetyError.focusChanged }
        if let windowID = snapshot.geometry.windowID, platform.activeWindowID() != windowID { throw SafetyError.focusChanged }
        guard try resolveGeometry(for: request) == snapshot.geometry else { throw SafetyError.unsupportedDisplay }
        guard try displayIDs() == snapshot.displayIDs else { throw SafetyError.unsupportedDisplay }
    }
    private func wait(_ milliseconds: Int, request: Envelope) throws {
        let end = ProcessInfo.processInfo.systemUptime + Double(milliseconds) / 1000
        while ProcessInfo.processInfo.systemUptime < end {
            try state.gate(request)
            Thread.sleep(forTimeInterval: min(0.01, max(0, end - ProcessInfo.processInfo.systemUptime)))
        }
        try state.gate(request)
    }
    // Launching or focusing an app can briefly make Window Server screenshot
    // capture unavailable after the input has already been posted. Retry only
    // the screenshot (never the input) for a short bounded interval; focus,
    // permission, geometry, and topology failures remain fail-closed.
    private func captureAfterInput(_ request: Envelope, geometry: DisplayGeometry) throws -> JSONValue {
        for attempt in 0..<3 {
            do {
                return try capture(request, geometry: geometry)
            } catch let error as SafetyError where error == .screenshotFailed && attempt < 2 {
                Thread.sleep(forTimeInterval: 0.05 * Double(attempt + 1))
            }
        }
        return try capture(request, geometry: geometry)
    }
    func execute(_ request: Envelope) -> ActionResult {
        var posted = false
        var pressedButton: CGMouseButton?
        defer { if let pressedButton { platform.releasePressedButton(pressedButton) } }
        do {
            let snapshot = try state.begin(request); try target(request, geometry: snapshot.geometry)
            let plan = try ActionPlan(request.payload, geometry: snapshot.geometry)
            try checkInput(snapshot, request: request)
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
            let payload = try captureAfterInput(request, geometry: snapshot.geometry)
            return ActionResult(outcome: .executed, payload: payload, error: nil)
        } catch {
            if posted { state.end() } // never run queued work after ambiguous input
            return ActionResult(outcome: posted ? .unknown : .rejected, payload: .object([:]), error: error as? SafetyError ?? .inputUnavailable)
        }
    }
}
