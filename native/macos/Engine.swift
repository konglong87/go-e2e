import Foundation

struct ActionResult {
    let outcome: Outcome
    let payload: JSONValue
    let error: SafetyError?
}
final class Engine {
    let state = SafetyState()
    private let platform: DesktopPlatform
    init(platform: DesktopPlatform) { self.platform = platform }

    func requestPermissions() -> JSONValue {
        platform.requestPermissions()
        return readiness()
    }

    func readiness() -> JSONValue {
        let geometry = try? platform.geometry()
        let capture = platform.captureAllowed(), input = platform.inputAllowed()
        return .object([
            "capture_readiness": .string(geometry == nil ? "unavailable" : (capture ? "ready" : "permission_required")),
            "input_readiness": .string(geometry == nil ? "unavailable" : (input ? "ready" : "permission_required")),
            "permission_state": .string(capture && input ? "approved" : "required"),
            "focus_state": .string(platform.focus() == nil ? "unavailable" : "focused"),
            "image_supported": .bool(geometry != nil), "supports_pause": .bool(true), "supports_stop": .bool(true),
            "coordinate_space": geometry?.coordinateSpace ?? .object(["origin":.string("top_left"),"unit":.string("pixels"),"width":.number(0),"height":.number(0),"scale_factor":.number(0)]),
            "actions": .array(ActionKind.allCases.map { .string($0.rawValue) })
        ])
    }
    private func target(_ request: Envelope, geometry: DisplayGeometry) throws {
        if let window = request.payload["window_id"]?.string, !window.isEmpty { throw SafetyError.unsupportedDisplay }
        if let display = request.payload["display_id"]?.string, !display.isEmpty && display != geometry.id { throw SafetyError.unsupportedDisplay }
    }
    private func capture(_ request: Envelope, geometry: DisplayGeometry) throws -> JSONValue {
        try state.gate(request)
        guard platform.captureAllowed(), try platform.geometry() == geometry else { throw SafetyError.screenshotFailed }
        let data = try platform.capture(geometry)
        try state.gate(request)
        guard try platform.geometry() == geometry else { throw SafetyError.unsupportedDisplay }
        return .object(["media_type": .string("image/png"), "data": .string(data.base64EncodedString()),
                        "width": .number(Double(geometry.width)), "height": .number(Double(geometry.height)),
                        "scale_factor": .number(geometry.scale), "display_id": .string(geometry.id)])
    }
    func observe(_ request: Envelope) -> ActionResult {
        do {
            try state.gate(request)
            let geometry = try platform.geometry(); try target(request, geometry: geometry)
            guard let id = request.payload["observation_id"]?.string, !id.isEmpty, id.utf8.count <= 256,
                  let focus = platform.focus(), let deadline = request.expires else { throw SafetyError.invalidAction }
            let payload = try capture(request, geometry: geometry)
            guard platform.focus() == focus else { throw SafetyError.focusChanged }
            try state.save(Snapshot(id: id, session: request.sessionID, geometry: geometry, focus: focus, expires: deadline), for: request)
            return ActionResult(outcome: .executed, payload: payload, error: nil)
        } catch { return ActionResult(outcome: .rejected, payload: .object([:]), error: error as? SafetyError ?? .screenshotFailed) }
    }
    private func checkInput(_ snapshot: Snapshot) throws {
        guard platform.inputAllowed(), platform.captureAllowed() else { throw SafetyError.permissionRequired }
        guard platform.focus() == snapshot.focus else { throw SafetyError.focusChanged }
        guard try platform.geometry() == snapshot.geometry else { throw SafetyError.unsupportedDisplay }
    }
    private func wait(_ milliseconds: Int, request: Envelope) throws {
        let end = ProcessInfo.processInfo.systemUptime + Double(milliseconds) / 1000
        while ProcessInfo.processInfo.systemUptime < end {
            try state.gate(request)
            Thread.sleep(forTimeInterval: min(0.01, max(0, end - ProcessInfo.processInfo.systemUptime)))
        }
        try state.gate(request)
    }
    func execute(_ request: Envelope) -> ActionResult {
        var posted = false
        do {
            let snapshot = try state.begin(request); try target(request, geometry: snapshot.geometry)
            let plan = try ActionPlan(request.payload, geometry: snapshot.geometry)
            try checkInput(snapshot)
            for (index, operation) in plan.operations.enumerated() {
                if plan.doubleClick && index > 0 { try wait(60, request: request) }
                // Revalidate TCC/focus/display just before each pair. Stop only
                // waits for this short critical section, not the entire action.
                try checkInput(snapshot)
                try state.gate(request) {
                    try platform.post(operation); posted = true
                }
            }
            try wait(plan.waitMS, request: request)
            let payload = try capture(request, geometry: snapshot.geometry)
            return ActionResult(outcome: .executed, payload: payload, error: nil)
        } catch {
            if posted { state.end() } // never run queued work after ambiguous input
            return ActionResult(outcome: posted ? .unknown : .rejected, payload: .object([:]), error: error as? SafetyError ?? .inputUnavailable)
        }
    }
}
