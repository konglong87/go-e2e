import AppKit

// Build: swiftc -parse-as-library -framework AppKit window_fixture.swift -o window-fixture
// Run: window-fixture --output /absolute/path/state.json
// JSON geometry is in global CG top-left points (not backing pixels). Window
// width/height include the title bar; buttonScreenFrame is the NSButton bounds.
// frontmost means the application's foremost visible window while it is active;
// keyWindow is AppKit's isKeyWindow. Closed windows disappear from the array.
private enum Fixture {
    static let title = "Computer Target Fixture"
    static let refreshInterval: TimeInterval = 0.25
    static let minimumSize = NSSize(width: 360, height: 280)
    static let keys = ["A", "B"]
}

private struct ScreenFrame: Encodable {
    let x: CGFloat
    let y: CGFloat
    let width: CGFloat
    let height: CGFloat

    init(_ appKitFrame: NSRect, primaryTop: CGFloat) {
        x = appKitFrame.minX
        y = primaryTop - appKitFrame.maxY
        width = appKitFrame.width
        height = appKitFrame.height
    }
}

private struct WindowState: Encodable {
    let key: String
    let windowNumber: Int
    let width: CGFloat
    let height: CGFloat
    let clickCount: Int
    let frame: ScreenFrame
    let buttonScreenFrame: ScreenFrame
    let frontmost: Bool
    let keyWindow: Bool
}

private struct Snapshot: Encodable {
    let processID: Int32
    let coordinateSystem = "CG-global-top-left-points"
    let frontmost: Bool
    let windows: [WindowState]
}

@MainActor
private final class FixtureWindow {
    let key: String
    let window: NSWindow
    let button: NSButton
    let counter = NSTextField(labelWithString: "Clicks: 0")
    var clickCount = 0

    init(key: String, frame: NSRect, target: AnyObject, action: Selector) {
        self.key = key
        window = NSWindow(contentRect: frame,
                          styleMask: [.titled, .closable, .miniaturizable, .resizable],
                          backing: .buffered, defer: false)
        window.title = Fixture.title
        window.identifier = NSUserInterfaceItemIdentifier("fixture-window-\(key)")
        window.isReleasedWhenClosed = false
        window.isRestorable = false
        window.tabbingMode = .disallowed
        window.contentMinSize = Fixture.minimumSize

        let heading = NSTextField(labelWithString: "Window \(key)")
        heading.font = .systemFont(ofSize: 52, weight: .bold)
        heading.alignment = .center
        button = NSButton(title: "Click \(key)", target: target, action: action)
        button.identifier = NSUserInterfaceItemIdentifier("fixture-button-\(key)")
        button.bezelStyle = .regularSquare
        button.font = .systemFont(ofSize: 30, weight: .semibold)
        counter.font = .monospacedDigitSystemFont(ofSize: 26, weight: .medium)
        counter.alignment = .center

        let stack = NSStackView(views: [heading, button, counter])
        stack.orientation = .vertical
        stack.alignment = .centerX
        stack.spacing = 24
        stack.translatesAutoresizingMaskIntoConstraints = false
        let content = window.contentView!
        content.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.centerXAnchor.constraint(equalTo: content.centerXAnchor),
            stack.centerYAnchor.constraint(equalTo: content.centerYAnchor),
            button.widthAnchor.constraint(equalToConstant: 260),
            button.heightAnchor.constraint(equalToConstant: 80),
        ])
    }

    func snapshot(primaryTop: CGFloat, frontWindow: NSWindow?) -> WindowState {
        window.contentView?.layoutSubtreeIfNeeded()
        let buttonFrame = window.convertToScreen(button.convert(button.bounds, to: nil))
        return WindowState(key: key, windowNumber: window.windowNumber,
                           width: window.frame.width, height: window.frame.height,
                           clickCount: clickCount,
                           frame: ScreenFrame(window.frame, primaryTop: primaryTop),
                           buttonScreenFrame: ScreenFrame(buttonFrame, primaryTop: primaryTop),
                           frontmost: NSApp.isActive && window === frontWindow,
                           keyWindow: window.isKeyWindow)
    }
}

@MainActor
private final class FixtureDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate {
    private let output: URL
    private let encoder = JSONEncoder()
    private var windows: [FixtureWindow] = []
    private var timer: Timer?
    private var ready = false

    init(output: URL) {
        self.output = output
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSWindow.allowsAutomaticWindowTabbing = false
        guard let screen = NSScreen.screens.first else {
            fail("No display available")
        }
        let visible = screen.visibleFrame
        // Distinct sizes and a horizontal offset keep both buttons exposed on
        // ordinary displays while permitting a little overlap on small screens.
        let sizeA = NSSize(width: 480, height: 360)
        let sizeB = NSSize(width: 440, height: 330)
        let originA = NSPoint(x: visible.minX + 24, y: visible.maxY - sizeA.height - 60)
        let originB = NSPoint(x: max(originA.x + 180, visible.maxX - sizeB.width - 24),
                              y: visible.maxY - sizeB.height - 130)
        let frames = [NSRect(origin: originA, size: sizeA), NSRect(origin: originB, size: sizeB)]
        for (key, frame) in zip(Fixture.keys, frames) {
            let item = FixtureWindow(key: key, frame: frame, target: self, action: #selector(click(_:)))
            item.window.delegate = self
            windows.append(item)
            item.window.makeKeyAndOrderFront(nil)
        }
        ready = true
        NSApp.activate(ignoringOtherApps: true)
        publish()
        let refresh = Timer(timeInterval: Fixture.refreshInterval, target: self,
                            selector: #selector(refreshState), userInfo: nil, repeats: true)
        timer = refresh
        RunLoop.main.add(refresh, forMode: .common)
    }

    @objc private func click(_ sender: NSButton) {
        guard let item = windows.first(where: { $0.button === sender }) else { return }
        item.clickCount += 1
        item.counter.stringValue = "Clicks: \(item.clickCount)"
        publish()
    }

    @objc private func refreshState() { publish() }
    func windowDidBecomeKey(_ notification: Notification) { publish() }
    func windowDidResignKey(_ notification: Notification) { publish() }
    func applicationDidBecomeActive(_ notification: Notification) { publish() }
    func applicationDidResignActive(_ notification: Notification) { publish() }

    func windowWillClose(_ notification: Notification) {
        guard let closing = notification.object as? NSWindow else { return }
        windows.removeAll { $0.window === closing }
        publish()
        if windows.isEmpty { NSApp.terminate(nil) }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { true }
    func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply { .terminateNow }
    func applicationWillTerminate(_ notification: Notification) { timer?.invalidate() }

    private func publish() {
        guard ready, let primaryScreen = NSScreen.screens.first else { return }
        // NSScreen.main follows the key window and is NOT the global origin.
        // AppKit's first screen defines the primary bottom-left origin. Flip
        // about its top edge, never about the current window's display height.
        let primaryTop = primaryScreen.frame.maxY
        let frontWindow = NSApp.orderedWindows.first { $0.isVisible && !$0.isMiniaturized }
        let state = Snapshot(processID: ProcessInfo.processInfo.processIdentifier,
                             frontmost: NSApp.isActive,
                             windows: windows.map { $0.snapshot(primaryTop: primaryTop, frontWindow: frontWindow) })
        do {
            try encoder.encode(state).write(to: output, options: .atomic)
        } catch {
            fail("Cannot write state: \(error)")
        }
    }
}

private func fail(_ message: String) -> Never {
    FileHandle.standardError.write(Data("window-fixture: \(message)\n".utf8))
    exit(EXIT_FAILURE)
}

@main
private struct WindowFixtureMain {
    @MainActor static func main() {
        let args = CommandLine.arguments
        guard args.count == 3, args[1] == "--output", (args[2] as NSString).isAbsolutePath else {
            fail("Usage: window-fixture --output <absolute-json-path>")
        }
        let output = URL(fileURLWithPath: args[2])
        do {
            try FileManager.default.createDirectory(at: output.deletingLastPathComponent(),
                                                    withIntermediateDirectories: true,
                                                    attributes: [.posixPermissions: 0o700])
        } catch {
            fail("Cannot prepare output directory: \(error)")
        }
        let application = NSApplication.shared
        application.setActivationPolicy(.regular)
        let delegate = FixtureDelegate(output: output)
        application.delegate = delegate
        withExtendedLifetime(delegate) { application.run() }
    }
}
