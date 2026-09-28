import Foundation
import CoreGraphics
import Carbon.HIToolbox

// Constructs events only. Never posts input, prompts for TCC, or captures UI.
@main
struct PlatformTests {
    static var assertions = 0
    static func expect(_ condition: @autoclosure () -> Bool, _ label: String) {
        assertions += 1
        guard condition() else { fatalError("FAIL: \(label)") }
    }
    static func check(_ event: CGEvent, _ type: CGEventType, _ key: Int,
                      _ flags: CGEventFlags, _ label: String) {
        expect(event.type == type, "\(label): event type")
        expect(event.getIntegerValueField(.keyboardEventKeycode) == Int64(key), "\(label): keycode")
        expect(event.flags == flags, "\(label): flags")
    }
    static func main() throws {
        let commandTab = try KeyboardEventSequence.make(code: CGKeyCode(kVK_Tab), flags: .maskCommand)
        expect(commandTab.count == 4, "command-tab has explicit modifier press/release")
        check(commandTab[0], .flagsChanged, kVK_Command, .maskCommand, "command down")
        check(commandTab[1], .keyDown, kVK_Tab, .maskCommand, "tab down")
        check(commandTab[2], .keyUp, kVK_Tab, .maskCommand, "tab up while command held")
        check(commandTab[3], .flagsChanged, kVK_Command, [], "command released")

        // All 16 combinations, including plain key input. This checks each
        // supported modifier, cumulative flags, reverse release, and final []
        // without depending on the physical keyboard's current modifier state.
        let modifiers: [(Int, CGEventFlags)] = [
            (kVK_Control, .maskControl), (kVK_Option, .maskAlternate),
            (kVK_Shift, .maskShift), (kVK_Command, .maskCommand),
        ]
        for mask in 0..<(1 << modifiers.count) {
            let selected = modifiers.enumerated().filter { mask & (1 << $0.offset) != 0 }.map { $0.element }
            let flags = selected.reduce(CGEventFlags()) { $0.union($1.1) }
            let events = try KeyboardEventSequence.make(code: CGKeyCode(kVK_Tab), flags: flags)
            expect(events.count == selected.count * 2 + 2, "balanced sequence")
            var held: CGEventFlags = []
            var index = 0
            for (key, flag) in selected {
                held.insert(flag)
                check(events[index], .flagsChanged, key, held, "modifier down")
                index += 1
            }
            check(events[index], .keyDown, kVK_Tab, flags, "main down")
            check(events[index + 1], .keyUp, kVK_Tab, flags, "main up")
            index += 2
            for (key, flag) in selected.reversed() {
                held.remove(flag)
                check(events[index], .flagsChanged, key, held, "modifier up")
                index += 1
            }
            expect(events.last!.flags.isEmpty, "no modifier remains")

            // Fail every allocation position, especially modifier releases.
            // Construction must throw, never return a partially postable chord.
            for failure in events.indices {
                var allocations = 0
                do {
                    _ = try KeyboardEventSequence.make(code: CGKeyCode(kVK_Tab), flags: flags) { key, down in
                        defer { allocations += 1 }
                        if allocations == failure { return nil }
                        return CGEvent(keyboardEventSource: nil, virtualKey: key, keyDown: down)
                    }
                    expect(false, "allocation failure must throw")
                } catch SafetyError.inputUnavailable {
                    expect(allocations == failure + 1, "stop at failed allocation")
                }
            }
        }
        let escape = try KeyboardEventSequence.make(code: CGKeyCode(kVK_Escape), flags: [])
        expect(escape.count == 2, "unmodified escape remains a key pair")
        check(escape[0], .keyDown, kVK_Escape, [], "escape down")
        check(escape[1], .keyUp, kVK_Escape, [], "escape up")
        for button in [CGMouseButton.left, .right] {
            let release = PreparedMouseRelease()
            let start = CGPoint(x: -200, y: 80)
            let end = CGPoint(x: -75, y: 140)
            try release.arm(button: button, point: start)
            expect(release.take(button: button == .left ? .right : .left) == nil, "wrong button cannot consume release")
            release.update(point: end)
            let event = release.take(button: button)
            expect(event?.type == (button == .left ? .leftMouseUp : .rightMouseUp), "prepared release has correct button")
            expect(event?.location == end, "emergency release uses last CG point without AppKit flip")
            expect(release.take(button: button) == nil, "release consumed once")
            do {
                try release.arm(button: button, point: start, create: { _, _, _ in nil })
                expect(false, "release allocation must fail before down")
            } catch SafetyError.inputUnavailable { expect(release.take(button: button) == nil, "failed preparation owns no release") }
        }
        for kind in [ActionKind.click, .doubleClick, .rightClick] {
            let events = try MouseEventSequence.make(kind: kind, point: CGPoint(x: 10, y: 20), clickCount: 2)
            expect(events.count == 2, "mouse pair preallocated")
            expect(events[1].getIntegerValueField(.mouseEventClickState) == 2, "click count preserved")
            var allocation = 0
            do {
                _ = try MouseEventSequence.make(kind: kind, point: .zero, clickCount: 1) { type, point, button in
                    allocation += 1
                    return allocation == 2 ? nil : CGEvent(mouseEventSource: nil, mouseType: type, mouseCursorPosition: point, mouseButton: button)
                }
                expect(false, "up allocation failure must not return postable down")
            } catch SafetyError.inputUnavailable { expect(allocation == 2, "mouse construction stops on failed release") }
        }
        print("PASS: \(assertions) platform event assertions (no events posted)")
    }
}
