package io.github.leonov792.iothome

import android.os.Handler
import android.os.Looper
import com.google.gson.Gson
import com.google.gson.annotations.SerializedName
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

data class Device(
    val id: String,
    val name: String,
    val type: String,
    val status: String,
    val room: String = "",
    val state: Map<String, Any> = emptyMap(),
    @SerializedName("owner_id") val ownerId: String = "",
    @SerializedName("created_at") val createdAt: String = "",
    @SerializedName("last_seen") val lastSeen: String = ""
) {
    val on: Boolean get() = state["on"] == true || state["on"] == true.toString()
}

data class Telemetry(
    val id: Long = 0,
    @SerializedName("device_id") val deviceId: String = "",
    val ts: String = "",
    val payload: Map<String, Double> = emptyMap()
)

data class LivePoint(
    @SerializedName("device_id") val deviceId: String = "",
    val ts: Long = 0,
    val temp: Double = 0.0,
    val humidity: Double = 0.0,
    val battery: Double = 0.0
)

interface Api {
    @POST("/api/v1/auth/login")
    suspend fun login(@Body body: Map<String, String>): Map<String, String>

    @POST("/api/v1/auth/register")
    suspend fun register(@Body body: Map<String, String>): Map<String, String>

    @GET("/api/v1/devices")
    suspend fun devices(): List<Device>

    @POST("/api/v1/devices")
    suspend fun createDevice(@Body body: Map<String, Any>): Device

    @DELETE("/api/v1/devices/{id}")
    suspend fun deleteDevice(@Path("id") id: String)

    @POST("/api/v1/devices/{id}/command")
    suspend fun command(@Path("id") id: String, @Body body: Map<String, Any>): Map<String, String>

    @GET("/api/v1/devices/{id}/telemetry")
    suspend fun telemetry(@Path("id") id: String): List<Telemetry>
}

object TokenHolder {
    var token: String? = null
}

object ApiClient {
    private val okHttp = OkHttpClient.Builder()
        .addInterceptor { chain ->
            val req = chain.request().newBuilder().apply {
                TokenHolder.token?.let { header("Authorization", "Bearer $it") }
            }.build()
            chain.proceed(req)
        }
        .build()

    val api: Api = Retrofit.Builder()
        .baseUrl(BuildConfig.API_BASE_URL)
        .client(okHttp)
        .addConverterFactory(GsonConverterFactory.create())
        .build()
        .create(Api::class.java)
}

// живая подписка на телеметрию через гейтвей
class TelemetryClient(
    private val deviceId: String,
    private val onPoint: (LivePoint) -> Unit
) {
    private val client = OkHttpClient()
    private val mainHandler = Handler(Looper.getMainLooper())
    private var ws: WebSocket? = null

    fun connect() {
        val req = Request.Builder().url(BuildConfig.WS_BASE_URL + "ws/dashboard").build()
        ws = client.newWebSocket(req, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                webSocket.send("""{"type":"subscribe","device_id":"$deviceId"}""")
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                // okhttp приходит с фонового потока, состояние compose трогаем только на main
                runCatching {
                    Gson().fromJson(text, LivePoint::class.java)
                }.onSuccess { mainHandler.post { onPoint(it) } }
            }
        })
    }

    fun close() {
        ws?.close(1000, null)
    }
}
