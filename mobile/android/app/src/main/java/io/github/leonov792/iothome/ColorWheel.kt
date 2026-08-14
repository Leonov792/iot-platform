package io.github.leonov792.iothome

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.gestures.detectDragGestures
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import kotlin.math.PI
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.roundToInt
import kotlin.math.sin

// Цветовое колесо (RGB/HSV) для управления RGB-освещением.
// hue 0..360, saturation 0..1, brightness 0..1.
@Composable
fun ColorWheel(
    hue: Float,
    saturation: Float,
    brightness: Float,
    onHueChange: (Float) -> Unit,
    onSaturationChange: (Float) -> Unit,
    onBrightnessChange: (Float) -> Unit,
    modifier: Modifier = Modifier
) {
    val diameter = 260.dp
    val density = LocalDensity.current
    val ringWidthPx = with(density) { 40.dp.toPx() }
    val diameterPx = with(density) { diameter.toPx() }

    val selected = Color.hsv(hue, saturation, brightness)

    Column(modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        Box(
            Modifier
                .size(diameter)
                .pointerInput(Unit) {
                    detectDragGestures { change, _ ->
                        val center = diameterPx / 2f
                        val dx = change.position.x - center
                        val dy = change.position.y - center
                        var angle = Math.toDegrees(atan2(dy.toDouble(), dx.toDouble())).toFloat()
                        if (angle < 0f) angle += 360f
                        onHueChange(angle)
                    }
                }
        ) {
            Canvas(Modifier.fillMaxSize()) {
                val radius = size.minDimension / 2f
                // кольцо оттенка
                drawCircle(
                    brush = Brush.sweepGradient(
                        (0..360 step 30).map { Color.hsv(it.toFloat(), 1f, 1f) }
                    ),
                    radius = radius,
                    style = Stroke(width = ringWidthPx, cap = StrokeCap.Round)
                )
                // внутренняя область насыщенность/яркость
                drawCircle(
                    brush = Brush.radialGradient(
                        colors = listOf(Color.White, selected, Color.Black)
                    ),
                    radius = radius - ringWidthPx
                )
            }

            // маркер оттенка
            val knobRadius = diameterPx / 2f - ringWidthPx / 2f
            val rad = hue * PI.toFloat() / 180f
            val dx = knobRadius * cos(rad)
            val dy = knobRadius * sin(rad)
            Box(
                Modifier
                    .size(22.dp)
                    .offset {
                        IntOffset(
                            (dx - with(density) { 11.dp.toPx() }).roundToInt(),
                            (dy - with(density) { 11.dp.toPx() }).roundToInt()
                        )
                    }
            ) {
                Canvas(Modifier.fillMaxSize()) {
                    drawCircle(Color.hsv(hue, 1f, 1f))
                    drawCircle(Color.White, style = Stroke(width = 4f))
                }
            }
        }

        Text(
            "#%02X%02X%02X".format(
                (selected.red * 255).toInt(),
                (selected.green * 255).toInt(),
                (selected.blue * 255).toInt()
            ),
            style = MaterialTheme.typography.bodyMedium
        )

        Slider(
            value = saturation,
            onValueChange = onSaturationChange,
            modifier = Modifier.padding(horizontal = 16.dp)
        )
        Slider(
            value = brightness,
            onValueChange = onBrightnessChange,
            modifier = Modifier.padding(horizontal = 16.dp)
        )
    }
}
