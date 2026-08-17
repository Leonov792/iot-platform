package io.github.leonov792.iothome

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

// X-Ray AR (скелет): тепловая карта труб/кабелей поверх стен.
//
// ВАЖНО: полноценная реализация требует ARCore (com.google.ar:core) + сцену
// (io.github.sceneview:arsceneview). Здесь — каркас и вспомогательная функция
// цветовой карты, чтобы не тянуть ARCore-зависимости в стандартную сборку.
// Точки интеграции помечены TODO.

// heatmapColor: температура -> цвет (холод = синий, тепло = красный).
fun heatmapColor(temp: Float): Color {
    val t = temp.coerceIn(10f, 60f)
    val k = (t - 10f) / 50f // 0..1
    return Color.hsv(240f - 240f * k, 1f, 1f)
}

@Composable
fun XRayARView(
    mesh: String = "",   // JSON из GET /api/v1/mesh (грузится один раз)
    telemetry: Float = 22f // температура зоны (приходит по WS)
) {
    // TODO: инициализировать ARCore-сессию и SLAM-трекинг;
    // привязать зоны труб к якорям (розетки/рамы/решётки);
    // наложить градиентную текстуру на стены по heatmapColor(telemetry).
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(
                Brush.linearGradient(
                    listOf(Color(0xFF0000FF), heatmapColor(telemetry), Color(0xFFFF0000))
                )
            ),
        contentAlignment = Alignment.Center
    ) {
        Text(
            "AR-каркас (нужен ARCore). Текущая темп. зоны: $telemetry°C",
            color = Color.White,
            modifier = Modifier.padding(16.dp)
        )
    }
}
