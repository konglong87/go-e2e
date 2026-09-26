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
        print("PASS: \(assertions) platform event assertions (no events posted)")
    }
}
