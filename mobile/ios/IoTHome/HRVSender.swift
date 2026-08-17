import Foundation
import HealthKit

// Отправка HRV (вариабельность сердечного ритма) в бэкенд каждые 10 секунд —
// для предсказания фазы пробуждения («мягкий рассвет»).
final class HRVSender {
    private let base: String
    private let token: String
    private let store = HKHealthStore()
    private var timer: Timer?

    init(base: String, token: String) {
        self.base = base
        self.token = token
    }

    func start() {
        // HealthKit-доступ запрашиваем один раз
        let type = HKObjectType.quantityType(forIdentifier: .heartRateVariabilitySDNN)!
        store.requestAuthorization(toShare: nil, read: [type]) { _, _ in }

        timer = Timer.scheduledTimer(withTimeInterval: 10, repeats: true) { [weak self] _ in
            self?.send()
        }
    }

    func stop() {
        timer?.invalidate()
        timer = nil
    }

    private func send() {
        latestHRV { [weak self] hrv in
            guard let self, hrv > 0 else { return }
            var req = URLRequest(url: URL(string: self.base + "/api/v1/health/hrv")!)
            req.httpMethod = "POST"
            req.setValue("Bearer \(self.token)", forHTTPHeaderField: "Authorization")
            req.setValue("application/json", forHTTPHeaderField: "Content-Type")
            req.httpBody = try? JSONSerialization.data(withJSONObject: ["value": hrv])
            URLSession.shared.dataTask(with: req).resume()
        }
    }

    // Читаем последний отсчёт HRV (SDNN) из HealthKit.
    private func latestHRV(completion: @escaping (Double) -> Void) {
        let type = HKQuantityType(.heartRateVariabilitySDNN)
        let sort = NSSortDescriptor(key: HKSampleSortIdentifierEndDate, ascending: false)
        let query = HKSampleQuery(sampleType: type, predicate: nil, limit: 1, sortDescriptors: [sort]) { _, samples, _ in
            guard let sample = samples?.first as? HKQuantitySample else {
                completion(0)
                return
            }
            completion(sample.quantity.doubleValue(for: HKUnit.secondUnit(with: .milli)))
        }
        store.execute(query)
    }
}
