package io.github.leonov792.iothome

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import com.google.ar.core.Config
import io.github.sceneview.ar.ARSceneView

// X-Ray AR (ARCore + SceneView 2.3): камера + SLAM-трекинг плоскостей + тепловая карта.
// Полноценное наложение 3D-теплокарты труб — поверх модели дома (GET /api/v1/mesh).

// heatmapColor: температура -> цвет (холод = синий, тепло = красный).
fun heatmapColor(temp: Float): Color {
    val t = temp.coerceIn(10f, 60f)
    val k = (t - 10f) / 50f // 0..1
    return Color.hsv(240f - 240f * k, 1f, 1f)
}

@Composable
fun XRayARView(
    zoneTemp: Float = 22f, // температура зоны (приходит по WS)
    modifier: Modifier = Modifier
) {
    Box(modifier = modifier.fillMaxSize()) {
        // Реальный ARCore: камера + детекция плоскостей (пол/стены)
        AndroidView(
            modifier = Modifier.fillMaxSize(),
            factory = { context ->
                ARSceneView(context).apply {
                    planeRenderer.isEnabled = true
                    configureSession { _, config: Config ->
                        config.planeFindingMode = Config.PlaneFindingMode.HORIZONTAL_AND_VERTICAL
                        config.lightEstimationMode = Config.LightEstimationMode.ENVIRONMENTAL_HDR
                    }
                }
            }
        )

        // TODO: привязать модель труб к SLAM-якорям из /api/v1/mesh и
        // красить материал по heatmapColor(telemetry). Здесь — цветовой индикатор.
        Box(
            modifier = Modifier
                .align(Alignment.TopCenter)
                .padding(top = 24.dp)
                .background(heatmapColor(zoneTemp).copy(alpha = 0.6f))
                .padding(horizontal = 12.dp, vertical = 6.dp)
        ) {
            Text("Темп. зоны: $zoneTemp°C", color = Color.White)
        }
    }
}
