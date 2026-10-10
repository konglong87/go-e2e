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

    static func matchingIndices(for target: NativeWindow, candidates: [AccessibleWindowDescriptor]) -> [Int] {
        guard target.isVisible, target.ownerPID > 0 else { return [] }
        return candidates.indices.filter { index in
            let candidate = candidates[index]
            return !candidate.minimized && framesMatch(target.frame, candidate.frame) &&
                (target.title.isEmpty || candidate.title == target.title)
        }
    }

    static func uniqueIndex(for target: NativeWindow, candidates: [AccessibleWindowDescriptor]) -> Int? {
        let matches = matchingIndices(for: target, candidates: candidates)
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
struct AccessibilityWindowActivationResult {
    let success: Bool
    let reason: String
    let axTrusted: Bool
    let axWindowCount: Int
    let geometryMatchCount: Int

    var diagnosticFields: [String: Any] {
        [
            "activation_succeeded": success,
            "activation_failure_reason": reason,
            "ax_trusted": axTrusted,
            "ax_window_count": axWindowCount,
            "ax_geometry_match_count": geometryMatchCount,
        ]
    }
}

enum AccessibilityWindowActivator {
    static let messagingTimeout: Float = 0.1
    static let maxWindows = 128

    static func raise(_ target: NativeWindow, permitted: () -> Bool, stillCurrent: () -> Bool) -> Bool {
        raiseDetailed(target, permitted: permitted, stillCurrent: stillCurrent).success
    }

    static func raiseDetailed(_ target: NativeWindow, permitted: () -> Bool, stillCurrent: () -> Bool) -> AccessibilityWindowActivationResult {
        func failure(_ reason: String, axTrusted: Bool = false, axWindowCount: Int = 0, geometryMatchCount: Int = 0) -> AccessibilityWindowActivationResult {
            AccessibilityWindowActivationResult(success: false, reason: reason, axTrusted: axTrusted,
                                                axWindowCount: axWindowCount, geometryMatchCount: geometryMatchCount)
        }
        guard permitted() else { return failure("permission_revoked") }
        let axTrusted = AXIsProcessTrusted()
        guard axTrusted else { return failure("ax_not_trusted") }
        guard target.ownerPID > 0 else { return failure("target_pid_invalid", axTrusted: axTrusted) }
        let appElement = AXUIElementCreateApplication(target.ownerPID)
        guard AXUIElementSetMessagingTimeout(appElement, messagingTimeout) == .success else {
            return failure("ax_app_timeout_setup_failed", axTrusted: axTrusted)
        }
        var count: CFIndex = 0
        guard permitted(), AXUIElementGetAttributeValueCount(appElement, kAXWindowsAttribute as CFString, &count) == .success,
              count > 0, count <= maxWindows else {
            return failure("ax_window_inventory_failed", axTrusted: axTrusted)
        }
        var value: CFArray?
        guard permitted(), AXUIElementCopyAttributeValues(appElement, kAXWindowsAttribute as CFString, 0, count, &value) == .success,
              let elements = value as? [AXUIElement] else {
            return failure("ax_window_values_failed", axTrusted: axTrusted, axWindowCount: Int(count))
        }
        // Do not drop unreadable windows: that could hide an ambiguous match.
        // Timeout is per AX object (not inherited from its application).
        var candidates: [(AXUIElement, AccessibleWindowDescriptor)] = []
        for element in elements {
            var pid: pid_t = 0
            guard permitted(), AXUIElementGetPid(element, &pid) == .success, pid == target.ownerPID,
                  AXUIElementSetMessagingTimeout(element, messagingTimeout) == .success,
                  let descriptor = descriptor(element, permitted: permitted) else {
                return failure("ax_window_unreadable", axTrusted: axTrusted, axWindowCount: Int(count))
            }
            candidates.append((element, descriptor))
        }
        let matches = WindowTargetSelector.matchingIndices(for: target, candidates: candidates.map { $0.1 })
        guard matches.count == 1 else {
            return failure(matches.isEmpty ? "ax_geometry_no_match" : "ax_geometry_ambiguous",
                           axTrusted: axTrusted, axWindowCount: Int(count), geometryMatchCount: matches.count)
        }
        guard permitted(), stillCurrent(), permitted(), let app = NSRunningApplication(processIdentifier: target.ownerPID) else {
            return failure("binding_changed_before_raise", axTrusted: axTrusted, axWindowCount: Int(count), geometryMatchCount: matches.count)
        }
        // Bring the same application window back to the active Space before
        // AXRaise. This is a same-PID/window recovery, not a target rebind.
        // Without activateAllWindows, a persisted window on another Space can
        // remain in the full Window Server inventory but never become the
        // frontmost on-screen target, causing launch/observe timeout.
        _ = app.activate(options: [.activateAllWindows])
        // Do not retry AX actions after timeout/unknown effects. The Engine will
        // require the requested ID to become frontmost before capture/input.
        guard permitted() else {
            return failure("permission_revoked_before_raise", axTrusted: axTrusted, axWindowCount: Int(count), geometryMatchCount: matches.count)
        }
        guard AXUIElementPerformAction(candidates[matches[0]].0, kAXRaiseAction as CFString) == .success else {
            return failure("ax_raise_failed", axTrusted: axTrusted, axWindowCount: Int(count), geometryMatchCount: matches.count)
        }
        return AccessibilityWindowActivationResult(success: true, reason: "", axTrusted: axTrusted,
                                                   axWindowCount: Int(count), geometryMatchCount: matches.count)
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
