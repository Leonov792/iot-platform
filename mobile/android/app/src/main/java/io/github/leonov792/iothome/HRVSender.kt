package io.github.leonov792.iothome

import android.content.Context
import androidx.health.connect.client.HealthConnectClient
import androidx.health.connect.client.records.HeartRateVariabilityRmssdRecord
import androidx.health.connect.client.request.ReadRecordsRequest
import androidx.health.connect.client.time.TimeRangeFilter
import com.google.gson.Gson
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.time.Instant

// Отправка HRV (вариабельность сердечного ритма) в бэкенд каждые 10 секунд.
// Читает реальные данные из Health Connect (HeartRateVariabilityRmssd).
class HRVSender(
    private val context: Context,
    private val baseUrl: String,
    private val token: String,
) {
    private val client = OkHttpClient()
    private var job: Job? = null

    fun start(scope: CoroutineScope) {
        stop()
        job = scope.launch(Dispatchers.IO) {
            while (true) {
                val hrv = readHRV()
                if (hrv > 0) send(hrv)
                delay(10_000)
            }
        }
    }

    fun stop() {
        job?.cancel()
        job = null
    }

    private suspend fun readHRV(): Double {
        return try {
            val hc = HealthConnectClient.getOrCreate(context)
            val response = hc.readRecords(
                ReadRecordsRequest(
                    recordType = HeartRateVariabilityRmssdRecord::class,
                    timeRangeFilter = TimeRangeFilter.after(Instant.now().minusSeconds(300)),
                    pageSize = 1
                )
            )
            response.records.lastOrNull()?.heartRateVariabilityMillis ?: 0.0
        } catch (e: Exception) {
            0.0
        }
    }

    private fun send(hrv: Double) {
        val body = Gson().toJson(mapOf("value" to hrv))
            .toRequestBody("application/json".toMediaType())

        val req = Request.Builder()
            .url(baseUrl + "/api/v1/health/hrv")
            .header("Authorization", "Bearer $token")
            .post(body)
            .build()

        runCatching { client.newCall(req).execute().close() }
    }
}
