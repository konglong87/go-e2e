import Foundation
import CoreGraphics

// Pure fake platform. This executable never constructs MacDesktop or posts a
// CGEvent, asks for permissions, focuses apps, or captures the actual desktop.
final class FakeDesktop: DesktopPlatform {
    var display = DisplayGeometry(id: "1", bounds: CGRect(x: 100, y: -50, width: 100, height: 50), width: 200, height: 100)
    var extraDisplays: [DisplayGeometry] = []
    var targetWindow: NativeWindow?
    var selectedWindowID: String?
    var activationCalls = 0
    var windowGeometryHook: (() -> Void)?
    var capturePermission = true
    var inputPermission = true
    var focused: Int32? = 42
    var failCapture = false
    var captureFailuresRemaining = 0
    var failGeometry = false
    var failPost = false
    var posts: [InputOperation] = []
    var postHook: (() -> Void)?
    var captureHook: (() -> Void)?
    var focusHook: (() -> Void)?
    func geometry() throws -> DisplayGeometry { if failGeometry { throw SafetyError.unsupportedDisplay }; return display }
    func geometries() throws -> [DisplayGeometry] { if failGeometry { throw SafetyError.unsupportedDisplay }; return [display] + extraDisplays }
    func windows() throws -> [NativeWindow] { targetWindow.map { [$0] } ?? [] }
    func windowGeometry(_ id: String) throws -> DisplayGeometry {
        windowGeometryHook?()
        guard let window = targetWindow, window.id == id else { throw SafetyError.unsupportedDisplay }
        return DisplayGeometry(id: window.displayID, bounds: window.frame, width: display.width, height: display.height, windowID: window.id)
    }
    func activateWindow(_ id: String) -> Bool { activationCalls += 1; guard targetWindow?.id == id else { return false }; selectedWindowID = id; focused = targetWindow?.ownerPID; return true }
    func activeWindowID() -> String? { selectedWindowID }
    func captureAllowed() -> Bool { capturePermission }
    func inputAllowed() -> Bool { inputPermission }
    func requestPermissions() { }
    func focus() -> Int32? { focusHook?(); return focused }
    func capture(_ g: DisplayGeometry) throws -> Data {
        if failCapture || captureFailuresRemaining > 0 {
            if captureFailuresRemaining > 0 { captureFailuresRemaining -= 1 }
            throw SafetyError.screenshotFailed
        }
        captureHook?(); return Data([1,2,3])
    }
    func post(_ operation: InputOperation) throws {
        if failPost { throw SafetyError.inputUnavailable }; posts.append(operation); postHook?()
    }
    func releasePressedButton(_ button: CGMouseButton) { posts.append(.mouseUp(.zero, button)) }
}
var assertions = 0
func expect(_ value: @autoclosure () -> Bool, _ label: String) {
    assertions += 1
    if !value() { fputs("FAIL: \(label)\n", stderr); exit(1) }
}
func expectThrows(_ label: String, _ body: () throws -> Void) {
    do { try body(); expect(false, label) } catch { assertions += 1 }
}
func request(_ command: String, payload: [String:JSONValue], session: String = "session", id: String = UUID().uuidString, seconds: Double = 5) -> Envelope {
    let formatter = ISO8601DateFormatter(); formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    var payload = payload; if payload["generation"] == nil { payload["generation"] = .number(0) }
    return Envelope(protocolVersion: protocolVersion, requestID: id, sessionID: session, actionID: id,
                    deadline: formatter.string(from: Date().addingTimeInterval(seconds)), command: command, payload: .object(payload))
}
func setup() -> (Engine,FakeDesktop) {
    let desktop = FakeDesktop(); let engine = Engine(platform: desktop)
    let observation = request("observe", payload: ["observation_id": .string("obs")])
    let result = engine.observe(observation)
    expect(result.outcome == .executed, "fake observation")
    return (engine,desktop)
}
func action(_ kind: String, _ extra: [String:JSONValue] = [:], seconds: Double = 5) -> Envelope {
    var p = extra; p["kind"] = .string(kind); p["observation_id"] = .string("obs")
    return request("execute", payload: p, seconds: seconds)
}

do {
    let desktop = FakeDesktop()
    desktop.extraDisplays = [DisplayGeometry(id: "2", bounds: CGRect(x: -100, y: 0, width: 100, height: 50), width: 200, height: 100)]
    let engine = Engine(platform: desktop)
    let ready = engine.readiness()
    expect(ready["displays"]?.array?.count == 2, "readiness exposes all displays")
    let secondarySpace = ready["displays"]?.array?[1]
    func number(_ value: JSONValue?) -> Double? {
        guard case .number(let value) = value else { return nil }
        return value
    }
    expect(number(secondarySpace?["bounds"]?["x"]) == -100 && number(secondarySpace?["bounds"]?["y"]) == 0,
           "readiness exposes global display origin")
    let result = engine.observe(request("observe", payload: ["observation_id":.string("secondary"), "display_id":.string("2")]))
    expect(result.outcome == .executed, "secondary display observation")
}

do {
    let desktop = FakeDesktop()
    desktop.extraDisplays = [DisplayGeometry(id: "2", bounds: CGRect(x: -100, y: 0, width: 100, height: 50), width: 200, height: 100)]
    let engine = Engine(platform: desktop)
    let observed = engine.observe(request("observe", payload: ["observation_id":.string("topology")]))
    expect(observed.outcome == .executed, "topology baseline observation")
    desktop.extraDisplays = []
    let result = engine.execute(action("click", ["x":.number(2), "y":.number(2)]))
    expect(result.outcome == .rejected && desktop.posts.isEmpty, "display topology change rejects stale action")
}

do {
    let desktop = FakeDesktop()
    desktop.targetWindow = NativeWindow(id: "window-9", title: "Fixture", ownerPID: 77, bundleID: "fixture.app", frame: desktop.display.bounds, displayID: desktop.display.id, isVisible: true, isFrontmost: false)
    let engine = Engine(platform: desktop)
    let observed = engine.observe(request("observe", payload: ["observation_id":.string("window-observation"), "window_id":.string("window-9")]))
    expect(observed.outcome == .executed, "window observation activates and captures target")
    let actionRequest = request("execute", payload: ["kind":.string("click"), "x":.number(2), "y":.number(2), "window_id":.string("window-9"), "observation_id":.string("window-observation")])
    let actionResult = engine.execute(actionRequest)
    expect(actionResult.outcome == .executed, "window-targeted click")
}

// Identity is not a mutable document title. No input or evidence read may
// reactivate a target and mask a user's focus change.
for change in ["title", "owner", "bundle", "frame", "closed", "focus-during-geometry", "focus-after-input"] {
    let desktop = FakeDesktop()
    let original = NativeWindow(id: "window-10", title: "Fixture", ownerPID: 77, bundleID: "fixture.app", frame: desktop.display.bounds, displayID: desktop.display.id, isVisible: true, isFrontmost: false)
    desktop.targetWindow = original
    let engine = Engine(platform: desktop)
    let observed = engine.observe(request("observe", payload: ["observation_id":.string("window-stale"), "window_id":.string(original.id)]))
    expect(observed.outcome == .executed, "window identity baseline: \(change)")
    let activations = desktop.activationCalls
    if change == "closed" {
        desktop.targetWindow = nil
    } else if change == "focus-during-geometry" {
        desktop.windowGeometryHook = { desktop.selectedWindowID = "other-window" }
    } else if change == "focus-after-input" {
        desktop.postHook = { desktop.selectedWindowID = "other-window" }
    } else {
        desktop.targetWindow = NativeWindow(id: original.id, title: change == "title" ? "Next page" : original.title,
            ownerPID: change == "owner" ? 78 : original.ownerPID, bundleID: change == "bundle" ? "other.app" : original.bundleID,
            frame: change == "frame" ? original.frame.offsetBy(dx: 1, dy: 0) : original.frame,
            displayID: original.displayID, isVisible: true, isFrontmost: false)
    }
    let result = engine.execute(request("execute", payload: ["kind":.string("move"), "x":.number(2), "y":.number(2), "window_id":.string(original.id), "observation_id":.string("window-stale")]))
    let expected: Outcome = change == "title" ? .executed : (change == "focus-after-input" ? .unknown : .rejected)
    expect(result.outcome == expected, "target outcome: \(change)")
    expect(desktop.posts.count == (change == "title" || change == "focus-after-input" ? 1 : 0), "target input count: \(change)")
    expect(desktop.activationCalls == activations, "Execute never activates window: \(change)")
}

// Safe integer conversion, overflow, bounds, NaN, infinity, fractions.
for n in [Double.nan, Double.infinity, -Double.infinity, 1e99, -1e99, 1.5, -1, 10001] {
    expect(JSONValue.number(n).integer(in: 0...10000) == nil, "invalid integer \(n)")
}
expect(JSONValue.number(Double(Int.max)).integer(in: 0...Int.max) == nil, "Int boundary rounding")
expect(JSONValue.number(10000).integer(in: 0...10000) == 10000, "inclusive bound")

let retina = FakeDesktop().display
let p = try retina.point(x: 198, y: 98)
expect(p.x == 199 && p.y == -1 && retina.scale == 2, "Retina pixels plus display origin")
expectThrows("off-display point") { _ = try retina.point(x: 200, y: 0) }
expectThrows("negative point") { _ = try retina.point(x: -1, y: 0) }
for payload in [
    ["kind": JSONValue.string("wait"), "duration_ms": .number(1e99)],
    ["kind": .string("scroll"), "delta_x": .number(1e99), "delta_y": .number(1)],
    ["kind": .string("click"), "x": .number(1.5), "y": .number(0)],
    ["kind": .string("type"), "text": .string(String(repeating: "x", count: maxTextUnits + 1))]
] { expectThrows("action numeric/text bound") { _ = try ActionPlan(.object(payload), geometry: retina) } }

let combo = try hotkey([.string("cmd"), .string("shift"), .string("c")])
expect(combo.code == 8 && combo.flags.contains(.maskCommand) && combo.flags.contains(.maskShift), "modifier flags")
for keys in [["cmd","NOT-A-KEY","c"],["cmd","command","c"],["a","b"],["cmd"],["c"],["cmd","shift"],["cmd","c","d"]] {
    expectThrows("invalid hotkey must not partially execute") { _ = try hotkey(keys.map{.string($0)}) }
}
let unicode = try ActionPlan(.object(["kind":.string("type"),"text":.string("A😀")]), geometry: retina)
expect(unicode.operations.count == 2, "Unicode scalar grouping")
if case .unicode(let units) = unicode.operations[1] { expect(units.count == 2, "surrogate pair preserved") } else { expect(false,"Unicode op") }
let click = try ActionPlan(.object(["kind":.string("double_click"),"x":.number(2),"y":.number(2)]), geometry: retina)
expect(click.operations.count == 2, "double click pair count")
if case .mouse(_, _, let count) = click.operations[1] { expect(count == 2, "double click state") }
let drag = try ActionPlan(.object(["kind":.string("drag"),"start_x":.number(10),"start_y":.number(10),"x":.number(190),"y":.number(90),"duration_ms":.number(160)]), geometry: retina)
expect(drag.operations.count >= 4, "drag has down, path, and up")
if case .mouseDown = drag.operations.first! { expect(true, "drag mouse down") } else { expect(false, "drag mouse down") }
if case .mouseUp = drag.operations.last! { expect(true, "drag mouse up") } else { expect(false, "drag mouse up") }
expect(drag.dragStepDelayMS > 0, "drag duration creates pacing")

do {
    let (engine, desktop) = setup()
    var postCount = 0
    desktop.postHook = {
        postCount += 1
        if postCount == 1 { desktop.focused = 7 }
    }
    let result = engine.execute(action("drag", ["start_x":.number(10), "start_y":.number(10), "x":.number(190), "y":.number(90), "duration_ms":.number(320)]))
    expect(result.outcome == .unknown && result.error == .focusChanged, "drag focus interruption is ambiguous")
    expect(desktop.posts.contains { if case .mouseUp = $0 { return true }; return false }, "interrupted drag releases mouse button")
}

for reason in ["permission","capture","focus","geometry","noevent","stopped","paused","expired","session","observation","display","window","generation"] {
    let (engine,desktop) = setup()
    var req = action("type", ["text":.string("abc")])
    switch reason {
    case "permission": desktop.inputPermission = false
    case "capture": desktop.capturePermission = false
    case "focus": desktop.focused = 7
    case "geometry": desktop.failGeometry = true
    case "noevent": desktop.failPost = true
    case "stopped": try engine.state.control(.stop, generation: 1)
    case "paused": try engine.state.control(.pause, generation: 1)
    case "expired": req = action("type", ["text":.string("abc")], seconds: -1)
    case "session": req = request("execute", payload:["kind":.string("type"),"text":.string("abc"),"observation_id":.string("obs")], session:"other")
    case "observation": req = request("execute", payload:["kind":.string("type"),"text":.string("abc"),"observation_id":.string("old")])
    case "display": req = action("type",["text":.string("abc"),"display_id":.string("2")])
    case "window": req = action("type",["text":.string("abc"),"window_id":.string("9")])
    default: req = action("type",["text":.string("abc"),"generation":.number(2)])
    }
    let result = engine.execute(req)
    expect(result.outcome == .rejected && desktop.posts.isEmpty, "pre-input rejection: \(reason)")
}

// A completed atomic action may activate another app. Its after-image must
// reflect the new focus; further input still requires a fresh observation.
let activatedFocus: Int32 = 7
let activationActions: [(String, [String:JSONValue])] = [
    ("hotkey", ["keys":.array([.string("command"),.string("tab")])]),
    ("key", ["key":.string("return")]),
    ("click", ["x":.number(2),"y":.number(2)]),
    ("type", ["text":.string("ab")]),
    ("double_click", ["x":.number(2),"y":.number(2)])
]
for (kind, payload) in activationActions {
    let (engine,desktop) = setup()
    let originalFocus = desktop.focused
    let operationCount = try ActionPlan(action(kind, payload).payload, geometry: desktop.display).operations.count
    desktop.postHook = {
        if desktop.posts.count == operationCount { desktop.focused = activatedFocus }
    }
    var capturedFocus: Int32?
    desktop.captureHook = { capturedFocus = desktop.focused }
    let result = engine.execute(action(kind, payload))
    expect(result.outcome == .executed && result.error == nil && desktop.posts.count == operationCount,
           "terminal focus switch accepted: \(kind)")
    expect(capturedFocus == activatedFocus && result.payload["data"]?.string == Data([1,2,3]).base64EncodedString(),
           "fresh after-image captured at activated focus: \(kind)")
    expect(engine.execute(action("key", ["key":.string("return")])).outcome == .rejected && desktop.posts.count == operationCount,
           "activation does not revive consumed observation: \(kind)")
    desktop.postHook = nil
    let nextObservation = "activated"
    expect(engine.observe(request("observe", payload:["observation_id":.string(nextObservation)])).outcome == .executed,
           "helper remains usable after activation: \(kind)")
    let next = request("execute", payload:["kind":.string("key"),"key":.string("return"),"observation_id":.string(nextObservation)])
    expect(engine.execute(next).outcome == .executed && desktop.posts.count == operationCount + 1,
           "fresh observation binds to activated focus: \(kind)")
    expect(engine.observe(request("observe", payload:["observation_id":.string(nextObservation)])).outcome == .executed,
           "observe activated focus again: \(kind)")
    desktop.focused = originalFocus
    let staleFocus = engine.execute(request("execute", payload:["kind":.string("key"),"key":.string("return"),"observation_id":.string(nextObservation)]))
    expect(staleFocus.outcome == .rejected && staleFocus.error == .focusChanged && desktop.posts.count == operationCount + 1,
           "new binding rejects old focus before dispatch: \(kind)")
}

// A Dock click waits for a delayed app activation before publishing after-image evidence.
do {
    let (engine, desktop) = setup()
    var focusReads = 0
    var capturedFocus: Int32?
    desktop.postHook = {
        desktop.focusHook = {
            focusReads += 1
            if focusReads >= 3 { desktop.focused = 7 }
        }
    }
    desktop.captureHook = { capturedFocus = desktop.focused }
    let result = engine.execute(action("click", ["x": .number(2), "y": .number(99)]))
    expect(result.outcome == .executed && desktop.posts.count == 1 && capturedFocus == 7,
           "Dock activation waits before publishing after-image")
}

// A transient post-input screenshot failure is retried without replaying input.
do {
    let (engine, desktop) = setup()
    desktop.captureFailuresRemaining = 2
    let result = engine.execute(action("click", ["x": .number(2), "y": .number(2)]))
    expect(result.outcome == .executed && desktop.posts.count == 1, "transient after-image capture retries without replay")
}

// Switching before dispatch must reject even an action intended to activate an app.
do {
    let (engine,desktop) = setup()
    desktop.focused = activatedFocus
    let result = engine.execute(action("hotkey", ["keys":.array([.string("command"),.string("tab")])]))
    expect(result.outcome == .rejected && result.error == .focusChanged && desktop.posts.isEmpty,
           "activation action rejects stale focus before dispatch")
}

// A switch between atomic operations must never send the rest into another app.
for (kind, payload) in activationActions where kind == "type" || kind == "double_click" {
    let (engine,desktop) = setup()
    desktop.postHook = { desktop.focused = activatedFocus }
    let result = engine.execute(action(kind, payload))
    expect(result.outcome == .unknown && result.error == .focusChanged && desktop.posts.count == 1,
           "focus switch between operations aborts: \(kind)")
    expectThrows("ambiguous multi-operation action cannot resume: \(kind)") { try engine.state.control(.resume,generation:1) }
}

// A new stable focus is acceptable; a change while capturing the after-image
// (including focus loss) is ambiguous and must still stop the helper.
for loseFocus in [false, true] {
    let (engine,desktop) = setup()
    let originalFocus = desktop.focused
    var captured = false
    desktop.postHook = { desktop.focused = activatedFocus }
    desktop.captureHook = {
        captured = true
        desktop.focused = loseFocus ? nil : originalFocus
    }
    let result = engine.execute(action("key", ["key":.string("return")]))
    expect(result.outcome == .unknown && result.error == .focusChanged && desktop.posts.count == 1,
           "focus changes during after-capture fail closed: \(loseFocus)")
    expect(captured && result.payload["data"] == nil, "inconsistent after-image is captured but not returned")
    expectThrows("inconsistent after-capture cannot resume") { try engine.state.control(.resume,generation:1) }
    expect(engine.observe(request("observe", payload:["observation_id":.string("after-failure")])).outcome == .rejected,
           "inconsistent after-capture prevents new binding")
}

// No focused app is not a valid new binding, even after a completed action.
do {
    let (engine,desktop) = setup()
    var captured = false
    desktop.postHook = { desktop.focused = nil }
    desktop.captureHook = { captured = true }
    let result = engine.execute(action("key", ["key":.string("return")]))
    expect(result.outcome == .unknown && result.error == .focusChanged && desktop.posts.count == 1 && !captured,
           "missing after-focus fails before capture")
    expectThrows("missing after-focus cannot resume") { try engine.state.control(.resume,generation:1) }
}

for reason in ["permission","capture","focus","geometry","screenshot","deadline"] {
    let (engine,desktop) = setup()
    desktop.postHook = {
        switch reason {
        case "permission": desktop.inputPermission = false
        case "capture": desktop.capturePermission = false
        case "focus": desktop.focused = activatedFocus
        case "geometry": desktop.failGeometry = true
        case "screenshot": desktop.failCapture = true
        default: Thread.sleep(forTimeInterval: 0.05)
        }
    }
    let result = engine.execute(action("type", ["text":.string(reason == "screenshot" ? "a" : "ab")], seconds: reason == "deadline" ? 0.03 : 5))
    expect(result.outcome == .unknown && desktop.posts.count == 1, "post-input uncertainty: \(reason)")
    expectThrows("unknown outcome cannot resume") { try engine.state.control(.resume,generation:1) }
}

do {
    let (engine,desktop) = setup()
    let req = action("hotkey",["keys":.array([.string("command"),.string("c")])])
    expect(engine.execute(req).outcome == .executed && desktop.posts.count == 1, "hotkey on fake platform")
    expect(engine.execute(req).outcome == .rejected, "consumed observation cannot replay")
    expect(engine.observe(request("observe",payload:["observation_id":.string("next")])).outcome == .executed, "fresh observe works")
    expect(engine.observe(request("observe",payload:["observation_id":.string("other")],session:"other")).outcome == .rejected, "session binding sticky")
}
// Pause/Resume invalidate both old observations and queued generation-0 work.
do {
    let (engine,desktop) = setup()
    try engine.state.control(.pause,generation:1); try engine.state.control(.resume,generation:2)
    expect(engine.execute(action("type",["text":.string("x")])).outcome == .rejected && desktop.posts.isEmpty, "stale queued generation")
    expect(engine.observe(request("observe",payload:["observation_id":.string("new"),"generation":.number(2)])).outcome == .executed, "observe after resume")
    expectThrows("out of order control") { try engine.state.control(.pause,generation:1) }
    try engine.state.control(.stop,generation:3)
    expectThrows("stop cannot resume") { try engine.state.control(.resume,generation:4) }
}
// A stop on the reader thread can interrupt a long wait on the worker.
do {
    let (engine,_) = setup()
    let done = DispatchSemaphore(value:0)
    let began = DispatchSemaphore(value:0)
    DispatchQueue.global().async {
        began.signal()
        let result = engine.execute(action("wait",["duration_ms":.number(5000)]))
        expect(result.outcome == .rejected, "interrupted wait never posted input")
        done.signal()
    }
    began.wait(); Thread.sleep(forTimeInterval:0.03)
    let start = Date(); try engine.state.control(.stop,generation:1)
    expect(done.wait(timeout:.now()+0.5) == .success && Date().timeIntervalSince(start)<0.5,"Stop not blocked by wait")
}
// Envelope deadlines, duplicate request IDs and malformed JSON never input.
do {
    let state = SafetyState()
    let req = request("readiness",payload:[:])
    try state.register(req)
    expectThrows("duplicate request ID") { try state.register(req) }
    expectThrows("expired deadline") { try state.register(request("execute",payload:[:],seconds:-1)) }
    expectThrows("unbounded deadline") { try state.register(request("execute",payload:[:],seconds:1000)) }
    let huge = Data("{\"x\":1e999}".utf8)
    expectThrows("overflow JSON number") { _ = try JSONDecoder().decode(JSONValue.self,from:huge) }
}

// Capture RPC deadlines and snapshot freshness are independent. Use an injected
// clock, not sleeps, to prove both RPC-expired/fresh and snapshot-expired cases.
for expiredSnapshot in [false, true] {
    var instant = Date()
    let desktop = FakeDesktop()
    let engine = Engine(platform: desktop, now: { instant })
    let observed = engine.observe(request("observe", payload: ["observation_id": .string("obs")], seconds: 1))
    expect(observed.outcome == .executed, "bounded capture RPC completed")
    expect(observed.payload["observation_expires_at"]?.string != nil, "helper reports actual snapshot expiry")
    instant = instant.addingTimeInterval(expiredSnapshot ? observationTTLSeconds + 1 : 2)
    let result = engine.execute(action("click", ["x": .number(2), "y": .number(2)], seconds: 60))
    if expiredSnapshot {
        expect(result.outcome == .rejected && desktop.posts.isEmpty, "expired snapshot rejected despite fresh action RPC")
    } else {
        expect(result.outcome == .executed && desktop.posts.count == 1, "snapshot survives completed capture RPC deadline")
    }
}

print("PASS: \(assertions) native safety assertions (fake platform; no real input/capture)")
