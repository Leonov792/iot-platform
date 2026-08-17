package io.github.leonov792.iothome

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

// Отправка HRV (вариабельность сердечного ритма) в бэкенд каждые 10 секунд —
// для предсказания фазы пробуждения («мягкий рассвет»).
class HRVSender(
    private val baseUrl: String,
    private val token: String,
    private val hrvProvider: () -> Double
) {
    private val client = OkHttpClient()
    private var job: Job? = null

    fun start(scope: CoroutineScope) {
        stop()
        job = scope.launch(Dispatchers.IO) {
            while (true) {
                send()
                delay(10_000)
            }
        }
    }

    fun stop() {
        job?.cancel()
        job = null
    }

    private fun send() {
        val hrv = hrvProvider()
        if (hrv <= 0) return

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

// HRV можно брать из носимого устройства (Google Fit / Health Connect / BLE-браслет).
// Пример провайдера — вернуть последний SDNN в мс или 0, если данных нет.
