import AppKit
import ApplicationServices
import CoreGraphics

struct AccessibleWindowDescriptor {
    let frame: CGRect
    let title: String
    let minimized: Bool
}

// Public AX has no public Window Server ID getter. Match within the already
// selected process, require geometry (never title alone), and reject ambiguity.
// Verify the actual Window Server ID after raising; a match is not authority.
enum WindowTargetSelector {
    static let geometryTolerance: CGFloat = 0.5

    static func uniqueIndex(for target: NativeWindow, candidates: [AccessibleWindowDescriptor]) -> Int? {
        guard target.isVisible, target.ownerPID > 0 else { return nil }
        let matches = candidates.indices.filter { index in
            let candidate = candidates[index]
            return !candidate.minimized && framesMatch(target.frame, candidate.frame) &&
                (target.title.isEmpty || candidate.title == target.title)
        }
        return matches.count == 1 ? matches.first : nil
    }

    static func framesMatch(_ lhs: CGRect, _ rhs: CGRect) -> Bool {
        let pairs = [(lhs.minX, rhs.minX), (lhs.minY, rhs.minY),
                     (lhs.width, rhs.width), (lhs.height, rhs.height)]
        return lhs.width > 0 && lhs.height > 0 && rhs.width > 0 && rhs.height > 0 &&
            pairs.allSatisfy { $0.0.isFinite && $0.1.isFinite && abs($0.0 - $0.1) <= geometryTolerance }
    }
}

// Keep the native AX adapter separate from the identity selection policy so
// matching and ambiguity are tested without touching the user's desktop.
enum AccessibilityWindowActivator {
    static let messagingTimeout: Float = 0.1
    static let maxWindows = 128

    static func raise(_ target: NativeWindow, permitted: () -> Bool, stillCurrent: () -> Bool) -> Bool {
        guard permitted(), AXIsProcessTrusted(), target.ownerPID > 0 else { return false }
        let appElement = AXUIElementCreateApplication(target.ownerPID)
        guard AXUIElementSetMessagingTimeout(appElement, messagingTimeout) == .success else { return false }
        var count: CFIndex = 0
        guard permitted(), AXUIElementGetAttributeValueCount(appElement, kAXWindowsAttribute as CFString, &count) == .success,
              count > 0, count <= maxWindows else { return false }
        var value: CFArray?
        guard permitted(), AXUIElementCopyAttributeValues(appElement, kAXWindowsAttribute as CFString, 0, count, &value) == .success,
              let elements = value as? [AXUIElement] else { return false }
        // Do not drop unreadable windows: that could hide an ambiguous match.
        // Timeout is per AX object (not inherited from its application).
        var candidates: [(AXUIElement, AccessibleWindowDescriptor)] = []
        for element in elements {
            var pid: pid_t = 0
            guard permitted(), AXUIElementGetPid(element, &pid) == .success, pid == target.ownerPID,
                  AXUIElementSetMessagingTimeout(element, messagingTimeout) == .success,
                  let descriptor = descriptor(element, permitted: permitted) else { return false }
            candidates.append((element, descriptor))
        }
        guard let index = WindowTargetSelector.uniqueIndex(for: target, candidates: candidates.map { $0.1 }),
              permitted(), stillCurrent(), permitted(), let app = NSRunningApplication(processIdentifier: target.ownerPID) else { return false }
        _ = app.activate(options: [])
        // Do not retry AX actions after timeout/unknown effects. The Engine will
        // require the requested ID to become frontmost before capture/input.
        guard permitted() else { return false }
        return AXUIElementPerformAction(candidates[index].0, kAXRaiseAction as CFString) == .success
    }

    private static func attribute(_ element: AXUIElement, _ name: String, permitted: () -> Bool) -> CFTypeRef? {
        var value: CFTypeRef?
        guard permitted(), AXUIElementCopyAttributeValue(element, name as CFString, &value) == .success else { return nil }
        return value
    }

    private static func descriptor(_ element: AXUIElement, permitted: () -> Bool) -> AccessibleWindowDescriptor? {
        guard let position = attribute(element, kAXPositionAttribute, permitted: permitted), CFGetTypeID(position) == AXValueGetTypeID(),
              let size = attribute(element, kAXSizeAttribute, permitted: permitted), CFGetTypeID(size) == AXValueGetTypeID(),
              let minimized = attribute(element, kAXMinimizedAttribute, permitted: permitted) as? Bool else { return nil }
        var point = CGPoint.zero
        var dimensions = CGSize.zero
        guard AXValueGetValue(position as! AXValue, .cgPoint, &point),
              AXValueGetValue(size as! AXValue, .cgSize, &dimensions) else { return nil }
        return AccessibleWindowDescriptor(frame: CGRect(origin: point, size: dimensions),
                                          title: attribute(element, kAXTitleAttribute, permitted: permitted) as? String ?? "",
                                          minimized: minimized)
    }
}
