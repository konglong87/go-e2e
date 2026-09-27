import AppKit
import ApplicationServices
import Carbon.HIToolbox
import CoreGraphics
import ImageIO
import UniformTypeIdentifiers

protocol DesktopPlatform {
    func geometry() throws -> DisplayGeometry
    func geometries() throws -> [DisplayGeometry]
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
}

struct MacDesktop: DesktopPlatform {
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
        guard captureAllowed(), let id = UInt32(geometry.id), let image = CGDisplayCreateImage(id),
              image.width == geometry.width, image.height == geometry.height else { throw SafetyError.screenshotFailed }
        let data = NSMutableData()
        guard let destination = CGImageDestinationCreateWithData(data, UTType.png.identifier as CFString, 1, nil) else { throw SafetyError.screenshotFailed }
        CGImageDestinationAddImage(destination, image, nil)
        guard CGImageDestinationFinalize(destination), data.length <= maxPNGBytes else { throw SafetyError.screenshotFailed }
        return data as Data
    }
    func post(_ operation: InputOperation) throws {
        switch operation {
        case .mouse(let kind, let point, let clickCount):
            let button: CGMouseButton = kind == .rightClick ? .right : .left
            let downType: CGEventType = kind == .move ? .mouseMoved : (button == .right ? .rightMouseDown : .leftMouseDown)
            guard let down = CGEvent(mouseEventSource: nil, mouseType: downType, mouseCursorPosition: point, mouseButton: button) else { throw SafetyError.inputUnavailable }
            down.flags = []
            down.setIntegerValueField(.mouseEventClickState, value: Int64(clickCount))
            down.post(tap: .cghidEventTap)
            if kind != .move {
                guard let up = CGEvent(mouseEventSource: nil, mouseType: button == .right ? .rightMouseUp : .leftMouseUp, mouseCursorPosition: point, mouseButton: button) else { throw SafetyError.inputUnavailable }
                up.flags = []; up.setIntegerValueField(.mouseEventClickState, value: Int64(clickCount)); up.post(tap: .cghidEventTap)
            }
        case .mouseDown(let point, let button):
            let type: CGEventType = button == .right ? .rightMouseDown : .leftMouseDown
            guard let event = CGEvent(mouseEventSource: nil, mouseType: type, mouseCursorPosition: point, mouseButton: button) else { throw SafetyError.inputUnavailable }
            event.flags = []; event.post(tap: .cghidEventTap)
        case .mouseDrag(let point, let button):
            let type: CGEventType = button == .right ? .rightMouseDragged : .leftMouseDragged
            guard let event = CGEvent(mouseEventSource: nil, mouseType: type, mouseCursorPosition: point, mouseButton: button) else { throw SafetyError.inputUnavailable }
            event.flags = []; event.post(tap: .cghidEventTap)
        case .mouseUp(let point, let button):
            let type: CGEventType = button == .right ? .rightMouseUp : .leftMouseUp
            guard let event = CGEvent(mouseEventSource: nil, mouseType: type, mouseCursorPosition: point, mouseButton: button) else { throw SafetyError.inputUnavailable }
            event.flags = []; event.post(tap: .cghidEventTap)
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
        let type: CGEventType = button == .right ? .rightMouseUp : .leftMouseUp
        if let event = CGEvent(mouseEventSource: nil, mouseType: type, mouseCursorPosition: NSEvent.mouseLocation, mouseButton: button) {
            event.flags = []; event.post(tap: .cghidEventTap)
        }
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
