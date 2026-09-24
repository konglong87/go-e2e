import AppKit
import ApplicationServices
import CoreGraphics
import ImageIO
import UniformTypeIdentifiers

protocol DesktopPlatform {
    func geometry() throws -> DisplayGeometry
    func captureAllowed() -> Bool
    func inputAllowed() -> Bool
    func focus() -> Int32?
    func capture(_ geometry: DisplayGeometry) throws -> Data
    func post(_ operation: InputOperation) throws
}

struct MacDesktop: DesktopPlatform {
    func geometry() throws -> DisplayGeometry {
        var count: UInt32 = 0
        guard CGGetActiveDisplayList(0, nil, &count) == .success, count == 1 else { throw SafetyError.unsupportedDisplay }
        let id = CGMainDisplayID()
        guard let mode = CGDisplayCopyDisplayMode(id) else { throw SafetyError.unsupportedDisplay }
        let g = DisplayGeometry(id: String(id), bounds: CGDisplayBounds(id), width: mode.pixelWidth, height: mode.pixelHeight)
        guard g.valid else { throw SafetyError.unsupportedDisplay }; return g
    }
    func captureAllowed() -> Bool { CGPreflightScreenCaptureAccess() }
    func inputAllowed() -> Bool { AXIsProcessTrusted() && CGPreflightPostEventAccess() }
    func focus() -> Int32? { NSWorkspace.shared.frontmostApplication?.processIdentifier }
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
        // Allocate *both* events before posting either one; always release any
        // key/button we press. Modifiers are flags, never held physical keys.
        let down: CGEvent?
        let up: CGEvent?
        switch operation {
        case .mouse(let kind, let point, let clickCount):
            let button: CGMouseButton = kind == .rightClick ? .right : .left
            let downType: CGEventType = kind == .move ? .mouseMoved : (button == .right ? .rightMouseDown : .leftMouseDown)
            down = CGEvent(mouseEventSource: nil, mouseType: downType, mouseCursorPosition: point, mouseButton: button)
            if kind == .move { up = nil }
            else { up = CGEvent(mouseEventSource: nil, mouseType: button == .right ? .rightMouseUp : .leftMouseUp, mouseCursorPosition: point, mouseButton: button) }
            down?.setIntegerValueField(.mouseEventClickState, value: Int64(clickCount)); up?.setIntegerValueField(.mouseEventClickState, value: Int64(clickCount))
            if kind != .move && up == nil { throw SafetyError.inputUnavailable }
        case .unicode(let units):
            down = CGEvent(keyboardEventSource: nil, virtualKey: 0, keyDown: true)
            up = CGEvent(keyboardEventSource: nil, virtualKey: 0, keyDown: false)
            guard up != nil else { throw SafetyError.inputUnavailable }
            down?.keyboardSetUnicodeString(stringLength: units.count, unicodeString: units)
            up?.keyboardSetUnicodeString(stringLength: units.count, unicodeString: units)
            down?.flags = []; up?.flags = []
        case .key(let code, let flags):
            down = CGEvent(keyboardEventSource: nil, virtualKey: code, keyDown: true)
            up = CGEvent(keyboardEventSource: nil, virtualKey: code, keyDown: false)
            guard up != nil else { throw SafetyError.inputUnavailable }
            down?.flags = flags; up?.flags = flags
        case .scroll(let x, let y):
            down = CGEvent(scrollWheelEvent2Source: nil, units: .pixel, wheelCount: 2, wheel1: y, wheel2: x, wheel3: 0); up = nil
        }
        guard let down else { throw SafetyError.inputUnavailable }
        if case .mouse = operation { down.flags = []; up?.flags = [] }
        if case .scroll = operation { down.flags = [] }
        down.post(tap: .cghidEventTap); up?.post(tap: .cghidEventTap)
    }
}
