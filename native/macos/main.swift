import Foundation

// Reader/control stays independent of the serial desktop executor. Bounded
// outstanding work avoids retaining arbitrary action/text queues in memory.
let engine = Engine(platform: MacDesktop())
let responder = Responder()
let executor = DispatchQueue(label: "computer-helper.actions")
let slots = DispatchSemaphore(value: 2)
while let data = readFrame() {
    guard let request = try? JSONDecoder().decode(Envelope.self, from: data) else { break }
    do {
        try engine.state.register(request)
        guard let command = Command(rawValue: request.command) else { throw SafetyError.invalidEnvelope }
        switch command {
        case .pause, .resume, .stop, .shutdown:
            let generation = request.payload["generation"]?.integer(in: 0...Int(Int32.max)) ?? -1
            try engine.state.control(command, generation: generation)
            responder.send(request, outcome: .executed)
            if command == .shutdown { exit(0) }
        case .readiness: responder.send(request, outcome: .executed, result: engine.readiness())
        case .observe, .execute:
            guard slots.wait(timeout: .now()) == .success else { throw SafetyError.capacity }
            executor.async {
                defer { slots.signal() }
                let result = command == .observe ? engine.observe(request) : engine.execute(request)
                responder.send(request, outcome: result.outcome, result: result.payload, error: result.error)
            }
        }
    } catch { responder.send(request, outcome: .rejected, error: error as? SafetyError ?? .invalidEnvelope) }
}
engine.state.end()
exit(0)
