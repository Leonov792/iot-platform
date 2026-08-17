import CoreMotion
import Foundation

// «Физический Ctrl+Z»: детектор намерения на комбинированном жесте.
// Резкий shake (встряска) + немедленный поворот экраном вниз.
// Это исключает ложные срабатывания (телефон в кармане при беге).
final class UndoGestureDetector: ObservableObject {
    @Published private(set) var undoTriggered = false

    private let motion = CMMotionManager()
    private var shakeTimes: [Date] = []

    // shakeThreshold — порог ускорения (g) для встряски.
    private let shakeThreshold = 2.5
    // shakeWindow — окно, в котором несколько встрясок считаются одним жестом.
    private let shakeWindow: TimeInterval = 1.5

    func start() {
        guard motion.isDeviceMotionAvailable else { return }
        motion.deviceMotionUpdateInterval = 1.0 / 100.0 // 100 Гц
        motion.startDeviceMotionUpdates(to: .main) { [weak self] data, _ in
            guard let self, let data else { return }
            self.process(data)
        }
    }

    func stop() {
        motion.stopDeviceMotionUpdates()
    }

    func reset() {
        undoTriggered = false
    }

    private func process(_ data: CMDeviceMotion) {
        let a = data.userAcceleration // ускорение без гравитации
        let magnitude = sqrt(a.x * a.x + a.y * a.y + a.z * a.z)

        guard magnitude > shakeThreshold else { return }

        let now = Date()
        shakeTimes.append(now)
        shakeTimes = shakeTimes.filter { now.timeIntervalSince($0) < shakeWindow }

        // экран вниз: вектор гравитации направлен вниз относительно устройства
        // (+z смотрит из экрана, поэтому лицом вниз -> z > 0.7)
        let faceDown = data.gravity.z > 0.7

        // как минимум две встряски + лицом вниз = осознанное намерение
        if faceDown && shakeTimes.count >= 2 {
            undoTriggered = true
            shakeTimes.removeAll()
        }
    }
}
