import AppKit
import ApplicationServices
import Carbon.HIToolbox
import CoreGraphics
import ImageIO
import UniformTypeIdentifiers

protocol DesktopPlatform {
    func geometry() throws -> DisplayGeometry
    func geometries() throws -> [DisplayGeometry]
    func windows() throws -> [NativeWindow]
    func windowGeometry(_ id: String) throws -> DisplayGeometry
    func activateWindow(_ id: String, permitted: () -> Bool) -> Bool
    func activeWindowID() -> String?
    func captureAllowed() -> Bool
    func inputAllowed() -> Bool
    func requestPermissions()
    func focus() -> Int32?
    func capture(_ geometry: DisplayGeometry) throws -> Data
    func post(_ operation: InputOperation) throws
    func releasePressedButton(_ button: CGMouseButton)
}

extension DesktopPlatform {
    func geometries() throws -> [DisplayGeometry] { [try geometry()] }
    func windows() throws -> [NativeWindow] { [] }
    func windowGeometry(_ id: String) throws -> DisplayGeometry { throw SafetyError.unsupportedDisplay }
    func activateWindow(_ id: String, permitted: () -> Bool) -> Bool { false }
    func activeWindowID() -> String? { nil }
}

struct MacDesktop: DesktopPlatform {
    private let preparedRelease = PreparedMouseRelease()

    func geometry() throws -> DisplayGeometry {
        let all = try geometries()
        let mainID = String(CGMainDisplayID())
        guard let main = all.first(where: { $0.id == mainID }) ?? all.first else { throw SafetyError.unsupportedDisplay }
        return main
    }

    func geometries() throws -> [DisplayGeometry] {
        var count: UInt32 = 0
        guard CGGetActiveDisplayList(0, nil, &count) == .success, count > 0 else { throw SafetyError.unsupportedDisplay }
        var ids = [CGDirectDisplayID](repeating: 0, count: Int(count))
        var actual: UInt32 = 0
        guard CGGetActiveDisplayList(count, &ids, &actual) == .success, actual > 0 else { throw SafetyError.unsupportedDisplay }
        let result = try ids.prefix(Int(actual)).map { id -> DisplayGeometry in
            guard let mode = CGDisplayCopyDisplayMode(id) else { throw SafetyError.unsupportedDisplay }
            let geometry = DisplayGeometry(id: String(id), bounds: CGDisplayBounds(id), width: mode.pixelWidth, height: mode.pixelHeight)
            guard geometry.valid else { throw SafetyError.unsupportedDisplay }
            return geometry
        }
        return result
    }
    func windows() throws -> [NativeWindow] {
        let options: CGWindowListOption = [.optionOnScreenOnly, .excludeDesktopElements]
        guard let raw = CGWindowListCopyWindowInfo(options, kCGNullWindowID) as? [[String: Any]] else { return [] }
        let activeID = activeWindowID()
        let displays = try geometries()
        return raw.compactMap { entry in
            guard let number = (entry[kCGWindowNumber as String] as? NSNumber)?.uint32Value,
                  let pid = (entry[kCGWindowOwnerPID as String] as? NSNumber)?.int32Value,
                  pid > 0,
                  let boundsDict = entry[kCGWindowBounds as String] as? NSDictionary,
                  let bounds = CGRect(dictionaryRepresentation: boundsDict),
                  bounds.width > 0, bounds.height > 0 else { return nil }
            let layer = (entry[kCGWindowLayer as String] as? NSNumber)?.intValue ?? 0
            let alpha = (entry[kCGWindowAlpha as String] as? NSNumber)?.doubleValue ?? 1
            let title = (entry[kCGWindowName as String] as? String) ?? ""
            let bundle = NSRunningApplication(processIdentifier: pid)?.bundleIdentifier ?? ""
            let displayID = displays.first(where: { $0.bounds.intersects(bounds) })?.id ?? String(CGMainDisplayID())
            return NativeWindow(id: String(number), title: title, ownerPID: pid, bundleID: bundle,
                                frame: bounds, displayID: displayID, isVisible: layer == 0 && alpha > 0,
                                isFrontmost: activeID == String(number))
        }
    }

    func windowGeometry(_ id: String) throws -> DisplayGeometry {
        guard let window = try windows().first(where: { $0.id == id }),
              let windowNumber = UInt32(window.id),
              let image = CGWindowListCreateImage(.null, .optionIncludingWindow, CGWindowID(windowNumber), [.bestResolution, .boundsIgnoreFraming]),
              image.width > 0, image.height > 0 else { throw SafetyError.unsupportedDisplay }
        let geometry = DisplayGeometry(id: window.displayID, bounds: window.frame, width: image.width, height: image.height, windowID: window.id)
        guard geometry.valid else { throw SafetyError.unsupportedDisplay }
        return geometry
    }

    func activateWindow(_ id: String, permitted: () -> Bool) -> Bool {
        guard permitted() else { return false }
        let list: [NativeWindow]
        do { list = try windows() } catch { return false }
        guard let window = list.first(where: { $0.id == id }), window.ownerPID > 0 else { return false }
        if focus() == window.ownerPID && (window.isFrontmost || activeWindowID() == id) { return true }
        return AccessibilityWindowActivator.raise(window, permitted: permitted) {
            guard let current = try? self.windows().first(where: { $0.id == id }) else { return false }
            return current.matchesIdentity(window)
        }
    }

    func activeWindowID() -> String? {
        let options: CGWindowListOption = [.optionOnScreenOnly, .excludeDesktopElements]
        guard let windows = CGWindowListCopyWindowInfo(options, kCGNullWindowID) as? [[String: Any]] else { return nil }
        // AppKit may expose a separate title-bar/menu companion window above the
        // real content window. Prefer the first substantial layer-0 window so a
        // target's identity remains stable across those decorations.
        var fallback: String?
        for window in windows {
            guard (window[kCGWindowLayer as String] as? NSNumber)?.intValue == 0,
                  let number = (window[kCGWindowNumber as String] as? NSNumber)?.uint32Value else { continue }
            fallback = fallback ?? String(number)
            if let boundsDict = window[kCGWindowBounds as String] as? NSDictionary,
               let bounds = CGRect(dictionaryRepresentation: boundsDict),
               bounds.width > 100, bounds.height > 100 {
                return String(number)
            }
        }
        return fallback
    }

    func captureAllowed() -> Bool { CGPreflightScreenCaptureAccess() }
    func inputAllowed() -> Bool { AXIsProcessTrusted() && CGPreflightPostEventAccess() }
    func requestPermissions() {
        _ = CGRequestScreenCaptureAccess()
        let options = [kAXTrustedCheckOptionPrompt.takeUnretainedValue() as String: true] as CFDictionary
        _ = AXIsProcessTrustedWithOptions(options)
    }
    // NSWorkspace can report the helper host itself while a different app
    // owns the topmost on-screen window (notably for a non-activating helper
    // launched by a Wails host). Use the Window Server ordering as the
    // authoritative frontmost-app signal, with NSWorkspace as a fallback.
    func focus() -> Int32? {
        let options: CGWindowListOption = [.optionOnScreenOnly, .excludeDesktopElements]
        if let windows = CGWindowListCopyWindowInfo(options, kCGNullWindowID) as? [[String: Any]] {
            for window in windows {
                let layer = (window[kCGWindowLayer as String] as? NSNumber)?.intValue
                guard layer == 0,
                      let pid = (window[kCGWindowOwnerPID as String] as? NSNumber)?.int32Value,
                      pid > 0 else { continue }
                return pid
            }
        }
        return NSWorkspace.shared.frontmostApplication?.processIdentifier
    }
    func capture(_ geometry: DisplayGeometry) throws -> Data {
        guard captureAllowed() else { throw SafetyError.screenshotFailed }
        let image: CGImage?
        if let windowID = geometry.windowID, let number = UInt32(windowID) {
            image = CGWindowListCreateImage(.null, .optionIncludingWindow, CGWindowID(number), [.bestResolution, .boundsIgnoreFraming])
        } else if let displayID = UInt32(geometry.id) {
            image = CGDisplayCreateImage(displayID)
        } else { image = nil }
        guard let image, image.width == geometry.width, image.height == geometry.height else { throw SafetyError.screenshotFailed }
        let data = NSMutableData()
        guard let destination = CGImageDestinationCreateWithData(data, UTType.png.identifier as CFString, 1, nil) else { throw SafetyError.screenshotFailed }
        CGImageDestinationAddImage(destination, image, nil)
        guard CGImageDestinationFinalize(destination), data.length <= maxPNGBytes else { throw SafetyError.screenshotFailed }
        return data as Data
    }
    func post(_ operation: InputOperation) throws {
        switch operation {
        case .mouse(let kind, let point, let clickCount):
            let events = try MouseEventSequence.make(kind: kind, point: point, clickCount: clickCount)
            for event in events { event.post(tap: .cghidEventTap) }
        case .mouseDown(let point, let button):
            let type: CGEventType = button == .right ? .rightMouseDown : .leftMouseDown
            guard let event = CGEvent(mouseEventSource: nil, mouseType: type, mouseCursorPosition: point, mouseButton: button) else { throw SafetyError.inputUnavailable }
            // Never post down without a preallocated emergency up event.
            try preparedRelease.arm(button: button, point: point)
            event.flags = []; event.post(tap: .cghidEventTap)
        case .mouseDrag(let point, let button):
            let type: CGEventType = button == .right ? .rightMouseDragged : .leftMouseDragged
            guard let event = CGEvent(mouseEventSource: nil, mouseType: type, mouseCursorPosition: point, mouseButton: button) else { throw SafetyError.inputUnavailable }
            preparedRelease.update(point: point)
            event.flags = []; event.post(tap: .cghidEventTap)
        case .mouseUp(let point, let button):
            preparedRelease.update(point: point)
            guard let event = preparedRelease.take(button: button) else { throw SafetyError.inputUnavailable }
            event.post(tap: .cghidEventTap)
        case .unicode(let units):
            guard let down = CGEvent(keyboardEventSource: nil, virtualKey: 0, keyDown: true),
                  let up = CGEvent(keyboardEventSource: nil, virtualKey: 0, keyDown: false) else { throw SafetyError.inputUnavailable }
            down.keyboardSetUnicodeString(stringLength: units.count, unicodeString: units)
            up.keyboardSetUnicodeString(stringLength: units.count, unicodeString: units)
            down.flags = []; up.flags = []; down.post(tap: .cghidEventTap); up.post(tap: .cghidEventTap)
        case .key(let code, let flags):
            let events = try KeyboardEventSequence.make(code: code, flags: flags)
            // Keep press/release together inside Engine's existing input gate.
            for event in events { event.post(tap: .cghidEventTap) }
        case .scroll(let x, let y):
            guard let event = CGEvent(scrollWheelEvent2Source: nil, units: .pixel, wheelCount: 2, wheel1: y, wheel2: x, wheel3: 0) else { throw SafetyError.inputUnavailable }
            event.flags = []; event.post(tap: .cghidEventTap)
        }
    }

    func releasePressedButton(_ button: CGMouseButton) {
        // Last posted CG event coordinates; do not mix in NSEvent's flipped
        // AppKit space or allocate during an input failure.
        preparedRelease.take(button: button)?.post(tap: .cghidEventTap)
    }
}

// CGEvent.h specifies modifier down/up events as part of a complete keystroke.
// CGEventCreateKeyboardEvent produces flagsChanged for modifier key codes;
// setting flags on the main key's keyUp alone is not a modifier release.
// Pure construction allows tests to inspect every event without posting input.
enum KeyboardEventSequence {
    static let modifiers: [(code: CGKeyCode, flag: CGEventFlags)] = [
        (CGKeyCode(kVK_Control), .maskControl),
        (CGKeyCode(kVK_Option), .maskAlternate),
        (CGKeyCode(kVK_Shift), .maskShift),
        (CGKeyCode(kVK_Command), .maskCommand),
    ]

    static func make(code: CGKeyCode, flags: CGEventFlags,
                     create: (CGKeyCode, Bool) -> CGEvent? = {
                         CGEvent(keyboardEventSource: nil, virtualKey: $0, keyDown: $1)
                     }) throws -> [CGEvent] {
        var events: [CGEvent] = []
        var held: CGEventFlags = []
        let pressed = modifiers.filter { flags.contains($0.flag) }
        func append(_ key: CGKeyCode, down: Bool, flags: CGEventFlags) throws {
            guard let event = create(key, down) else { throw SafetyError.inputUnavailable }
            event.flags = flags
            events.append(event)
        }
        for modifier in pressed {
            held.insert(modifier.flag)
            try append(modifier.code, down: true, flags: held)
        }
        try append(code, down: true, flags: flags)
        try append(code, down: false, flags: flags)
        for modifier in pressed.reversed() {
            held.remove(modifier.flag)
            try append(modifier.code, down: false, flags: held)
        }
        return events
    }
}

// Construction is independently testable without posting any OS input.
enum MouseEventSequence {
    typealias Factory = (CGEventType, CGPoint, CGMouseButton) -> CGEvent?
    static func make(kind: ActionKind, point: CGPoint, clickCount: Int,
                     create: Factory = { CGEvent(mouseEventSource: nil, mouseType: $0, mouseCursorPosition: $1, mouseButton: $2) }) throws -> [CGEvent] {
        let button: CGMouseButton = kind == .rightClick ? .right : .left
        let down: CGEventType = kind == .move ? .mouseMoved : (button == .right ? .rightMouseDown : .leftMouseDown)
        let types: [CGEventType] = kind == .move ? [down] : [down, button == .right ? .rightMouseUp : .leftMouseUp]
        return try types.map { type in
            guard let event = create(type, point, button) else { throw SafetyError.inputUnavailable }
            event.flags = []
            event.setIntegerValueField(.mouseEventClickState, value: Int64(clickCount))
            return event
        }
    }
}

// Owned by the serial native executor. This is cooperative cleanup only;
// it cannot survive SIGKILL and is not an independent crash-release guard.
final class PreparedMouseRelease {
    private var event: CGEvent?
    private var button: CGMouseButton?

    func arm(button: CGMouseButton, point: CGPoint,
             create: MouseEventSequence.Factory = { CGEvent(mouseEventSource: nil, mouseType: $0, mouseCursorPosition: $1, mouseButton: $2) }) throws {
        guard event == nil else { throw SafetyError.invalidAction }
        let type: CGEventType = button == .right ? .rightMouseUp : .leftMouseUp
        guard let up = create(type, point, button) else { throw SafetyError.inputUnavailable }
        up.flags = []
        self.button = button; event = up
    }
    func update(point: CGPoint) { event?.location = point }
    func take(button: CGMouseButton) -> CGEvent? {
        guard self.button == button else { return nil }
        let up = event
        event = nil; self.button = nil
        return up
    }
}
