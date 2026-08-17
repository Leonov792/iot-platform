import ARKit
import SwiftUI

// X-Ray AR (ARKit): камера + SLAM-трекинг плоскостей + тепловая карта зоны.
// Полноценное наложение 3D-теплокарты труб делается поверх модели дома
// (GET /api/v1/mesh). Здесь — реальный AR-слой и цветовой оверлей.

// heatmapColor: температура -> цвет (холод = синий, тепло = красный).
func heatmapColor(temp: Float) -> UIColor {
    let t = min(max(temp, 10), 60)
    let k = (t - 10) / 50 // 0..1
    return UIColor(hue: CGFloat(0.66 - 0.66 * k), saturation: 1, brightness: 1, alpha: 0.8)
}

struct XRayARView: UIViewRepresentable {
    let zoneTemp: Float // температура зоны (обновляется по WS)

    func makeUIView(context: Context) -> ARSCNView {
        let sceneView = ARSCNView()
        sceneView.delegate = context.coordinator
        sceneView.showsStatistics = true

        let config = ARWorldTrackingConfiguration()
        // SLAM: ищем горизонтальные плоскости (пол) и вертикальные (стены)
        config.planeDetection = [.horizontal, .vertical]
        sceneView.session.run(config)

        return sceneView
    }

    func updateUIView(_ uiView: ARSCNView, context: Context) {
        // обновляем цветовую подсветку при изменении температуры зоны
        context.coordinator.updateOverlay(temp: zoneTemp, in: uiView)
    }

    func makeCoordinator() -> Coordinator { Coordinator() }

    final class Coordinator: NSObject, ARSCNViewDelegate {
        private var overlayPlanes: [SCNNode] = []

        func updateOverlay(temp: Float, in sceneView: ARSCNView) {
            let color = heatmapColor(temp: temp)

            // Подсвечиваем найденные плоскости цветом зоны (упрощённая теплокарта).
            // TODO: привязать к якорям труб из /api/v1/mesh и красить по-сегментно.
            for node in overlayPlanes {
                node.geometry?.firstMaterial?.diffuse.contents = color
            }
        }

        // ARSCNViewDelegate: при обнаружении плоскости — красим её
        func renderer(_ renderer: SCNSceneRenderer, didAdd node: SCNNode, for anchor: ARAnchor) {
            guard let planeAnchor = anchor as? ARPlaneAnchor else { return }

            let plane = SCNPlane(width: CGFloat(planeAnchor.extent.x), height: CGFloat(planeAnchor.extent.z))
            plane.firstMaterial?.diffuse.contents = UIColor.white.withAlphaComponent(0.3)
            plane.firstMaterial?.isDoubleSided = true

            let planeNode = SCNNode(geometry: plane)
            planeNode.transform = SCNMatrix4MakeRotation(-Float.pi / 2, 1, 0, 0)
            node.addChildNode(planeNode)
            overlayPlanes.append(planeNode)
        }
    }
}
