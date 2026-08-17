import ARKit
import SwiftUI

// X-Ray AR (скелет): модель дома из API + тепловая карта поверх стен по телеметрии.
//
// ВАЖНО: это каркас для доработки на реальном устройстве (нужен ARKit + SLAM).
// Ключевые куски помечены TODO — их доводят под конкретную схему дома/зон.
final class XRayARView: UIViewRepresentable {
    let meshURL: URL // GET /api/v1/mesh (загружается один раз)
    let wsURL: URL   // WS телеметрии (лёгкий поток temp/ph)

    func makeUIView(context: Context) -> ARSCNView {
        let sceneView = ARSCNView()
        sceneView.delegate = context.coordinator
        sceneView.showsStatistics = true

        let config = ARWorldTrackingConfiguration()
        // SLAM: ищем горизонтальные плоскости (стены/пол)
        config.planeDetection = [.horizontal, .vertical]
        sceneView.session.run(config)

        context.coordinator.loadMesh(meshURL)
        context.coordinator.connect(wsURL)

        return sceneView
    }

    func updateUIView(_ uiView: ARSCNView, context: Context) {}

    func makeCoordinator() -> Coordinator { Coordinator() }

    final class Coordinator: NSObject, ARSCNViewDelegate {
        private var zones: [Zone] = []

        struct Zone: Codable {
            let deviceID: String
            let positions: [SIMD3<Float>] // упрощённо: точки сегмента трубы
        }

        func loadMesh(_ url: URL) {
            URLSession.shared.dataTask(with: url) { data, _, _ in
                guard let data else { return }
                // TODO: распарсить mesh/anchors/zones и привязать к SLAM-якорям
                // (ARAnchor на реперные точки: розетки, рамы, решётки)
            }.resume()
        }

        func connect(_ url: URL) {
            // TODO: подписаться на WS-телеметрию и красить зоны по temp
        }

        // heatmap: температура -> цвет (холод = синий, тепло = красный)
        func heatmapColor(temp: Float) -> UIColor {
            let t = min(max(temp, 10), 60)
            let k = (t - 10) / 50 // 0..1
            return UIColor(hue: CGFloat(0.66 - 0.66 * k), saturation: 1, brightness: 1, alpha: 0.8)
        }
    }
}
