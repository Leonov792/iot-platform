package io.github.leonov792.iothome

import android.content.Context
import android.hardware.Sensor
import android.hardware.SensorEvent
import android.hardware.SensorEventListener
import android.hardware.SensorManager
import kotlin.math.sqrt

// «Физический Ctrl+Z»: комбинированный жест — резкий shake + немедленный поворот
// экраном вниз. Исключает ложные срабатывания (телефон в кармане при беге).
class UndoGestureDetector(context: Context) : SensorEventListener {
    private val sensorManager =
        context.getSystemService(Context.SENSOR_SERVICE) as SensorManager
    private val linearAccel = sensorManager.getDefaultSensor(Sensor.TYPE_LINEAR_ACCELERATION)
    private val gravitySensor = sensorManager.getDefaultSensor(Sensor.TYPE_GRAVITY)

    var onUndo: (() -> Unit)? = null

    private val shakeTimes = ArrayDeque<Long>()
    private var lastGravityZ = 0f

    private val shakeThreshold = 2.5f // g
    private val shakeWindowMs = 1500L

    fun start() {
        linearAccel?.let {
            sensorManager.registerListener(this, it, SensorManager.SENSOR_DELAY_GAME)
        }
        gravitySensor?.let {
            sensorManager.registerListener(this, it, SensorManager.SENSOR_DELAY_GAME)
        }
    }

    fun stop() {
        sensorManager.unregisterListener(this)
    }

    override fun onSensorChanged(event: SensorEvent) {
        when (event.sensor.type) {
            Sensor.TYPE_LINEAR_ACCELERATION -> {
                val x = event.values[0]
                val y = event.values[1]
                val z = event.values[2]
                val magnitude = sqrt(x * x + y * y + z * z)

                if (magnitude > shakeThreshold) {
                    val now = System.currentTimeMillis()
                    shakeTimes.addLast(now)
                    while (shakeTimes.isNotEmpty() && now - shakeTimes.first() > shakeWindowMs) {
                        shakeTimes.removeFirst()
                    }
                    // экран вниз: gravity z > +7 м/с^2 (TYPE_GRAVITY в м/с^2)
                    if (shakeTimes.size >= 2 && lastGravityZ > 7.0f) {
                        shakeTimes.clear()
                        onUndo?.invoke()
                    }
                }
            }

            Sensor.TYPE_GRAVITY -> {
                lastGravityZ = event.values[2]
            }
        }
    }

    override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) = Unit
}
