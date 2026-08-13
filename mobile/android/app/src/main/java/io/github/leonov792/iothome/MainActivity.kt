package io.github.leonov792.iothome

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme { App() }
        }
    }
}

private sealed interface Screen {
    data object Login : Screen
    data object Devices : Screen
    data class Detail(val id: String) : Screen
}

@Composable
private fun App() {
    var screen by remember { mutableStateOf<Screen>(Screen.Login) }

    when (val s = screen) {
        Screen.Login -> LoginScreen(onDone = { screen = Screen.Devices })
        Screen.Devices -> DevicesScreen(
            onOpen = { screen = Screen.Detail(it) },
            onLogout = { screen = Screen.Login }
        )
        is Screen.Detail -> DeviceDetailScreen(s.id, onBack = { screen = Screen.Devices })
    }
}

@Composable
private fun LoginScreen(onDone: () -> Unit) {
    val scope = rememberCoroutineScope()
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var register by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(false) }

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Center
    ) {
        Text("Умный дом", style = MaterialTheme.typography.headlineMedium)
        Text(if (register) "Регистрация" else "Вход", color = Color.Gray)
        Spacer(Modifier.height(24.dp))

        OutlinedTextField(email, { email = it }, label = { Text("Почта") }, modifier = Modifier.fillMaxWidth())
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            password,
            { password = it },
            label = { Text("Пароль") },
            modifier = Modifier.fillMaxWidth()
        )
        Spacer(Modifier.height(8.dp))

        error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        Button(
            onClick = {
                loading = true
                scope.launch {
                    runCatching {
                        val body = mapOf("email" to email, "password" to password)
                        if (register) ApiClient.api.register(body) else ApiClient.api.login(body)
                    }.onSuccess { resp ->
                        TokenHolder.token = resp["token"]
                        onDone()
                    }.onFailure { e ->
                        error = e.message ?: "ошибка"
                    }
                    loading = false
                }
            },
            enabled = !loading,
            modifier = Modifier.fillMaxWidth()
        ) {
            Text(if (register) "Создать аккаунт" else "Войти")
        }

        OutlinedButton(onClick = { register = !register }, modifier = Modifier.fillMaxWidth()) {
            Text(if (register) "Уже есть? Войти" else "Нет аккаунта? Регистрация")
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DevicesScreen(onOpen: (String) -> Unit, onLogout: () -> Unit) {
    val scope = rememberCoroutineScope()
    var devices by remember { mutableStateOf<List<Device>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var adding by remember { mutableStateOf(false) }
    var name by remember { mutableStateOf("") }
    var type by remember { mutableStateOf("light") }
    var room by remember { mutableStateOf("") }
    var id by remember { mutableStateOf("") }

    fun reload() {
        scope.launch {
            runCatching { ApiClient.api.devices() }
                .onSuccess { devices = it }
            loading = false
        }
    }

    LaunchedEffect(Unit) { reload() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Устройства") },
                actions = { OutlinedButton(onClick = { TokenHolder.token = null; onLogout() }) { Text("Выйти") } }
            )
        }
    ) { pad ->
        Column(Modifier.fillMaxSize().padding(pad).padding(16.dp)) {
            if (loading) {
                CircularProgressIndicator(Modifier.align(Alignment.CenterHorizontally))
            } else {
                LazyColumn(Modifier.weight(1f)) {
                    items(devices, key = { it.id }) { d ->
                        DeviceRow(d) { onOpen(d.id) }
                        HorizontalDivider()
                    }
                }

                if (adding) {
                    Spacer(Modifier.height(8.dp))
                    Column {
                        OutlinedTextField(name, { name = it }, label = { Text("Название") }, modifier = Modifier.fillMaxWidth())
                        Row {
                            OutlinedTextField(type, { type = it }, label = { Text("Тип") }, modifier = Modifier.weight(1f))
                            Spacer(Modifier.width(8.dp))
                            OutlinedTextField(room, { room = it }, label = { Text("Комната") }, modifier = Modifier.weight(1f))
                        }
                        OutlinedTextField(id, { id = it }, label = { Text("ID (lamp-1)") }, modifier = Modifier.fillMaxWidth())
                        Button(onClick = {
                            scope.launch {
                                runCatching {
                                    ApiClient.api.createDevice(
                                        mapOf("id" to id, "name" to name, "type" to type, "room" to room)
                                    )
                                }.onSuccess {
                                    adding = false; name = ""; id = ""; room = ""; reload()
                                }
                            }
                        }) { Text("Создать") }
                    }
                } else {
                    Spacer(Modifier.height(8.dp))
                    Button(onClick = { adding = true }, modifier = Modifier.fillMaxWidth()) { Text("+ Добавить") }
                }
            }
        }
    }
}

@Composable
private fun DeviceRow(device: Device, onClick: () -> Unit) {
    val scope = rememberCoroutineScope()

    Card(onClick = onClick, modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
        Row(Modifier.padding(16.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(device.name, style = MaterialTheme.typography.titleMedium)
                Text("${typeLabel(device.type)} · ${device.room.ifBlank { "без комнаты" }}", color = Color.Gray)
            }

            when (device.type) {
                "light", "plug" -> {
                    Button(onClick = {
                        scope.launch {
                            runCatching {
                                ApiClient.api.command(device.id, mapOf("action" to if (device.on) "off" else "on"))
                            }
                        }
                    }) {
                        Text(if (device.on) "Выкл" else "Вкл")
                    }
                }
                else -> Text(device.status, color = if (device.status == "online") Color(0xFF16A34A) else Color.Gray)
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DeviceDetailScreen(id: String, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    var device by remember { mutableStateOf<Device?>(null) }
    var temps by remember { mutableStateOf<List<Double>>(emptyList()) }
    var latest by remember { mutableStateOf<LivePoint?>(null) }
    var target by remember { mutableStateOf("22") }

    fun reload() {
        scope.launch {
            runCatching { ApiClient.api.devices() }
                .onSuccess { list -> device = list.find { it.id == id } }
            runCatching { ApiClient.api.telemetry(id) }
                .onSuccess { rows -> temps = rows.mapNotNull { it.payload["temp"] } }
        }
    }

    fun send(action: String, value: Double? = null) {
        scope.launch {
            val body = mutableMapOf<String, Any>("action" to action)
            value?.let { body["value"] = it }
            runCatching { ApiClient.api.command(id, body) }.onSuccess { reload() }
        }
    }

    LaunchedEffect(Unit) { reload() }

    DisposableEffect(id) {
        val client = TelemetryClient(id) { p ->
            latest = p
            temps = temps + p.temp
        }
        client.connect()
        onDispose { client.close() }
    }

    Scaffold(topBar = { TopAppBar(title = { Text(device?.name ?: "...") }, navigationIcon = { Button(onClick = onBack) { Text("←") } }) }) { pad ->
        Column(Modifier.fillMaxSize().padding(pad).padding(16.dp)) {
            Text(device?.room.orEmpty(), color = Color.Gray)

            Spacer(Modifier.height(8.dp))
            when (device?.type) {
                "light" -> Row(verticalAlignment = Alignment.CenterVertically) {
                    Button(onClick = { send(if (device?.on == true) "off" else "on") }) {
                        Text(if (device?.on == true) "Выключить" else "Включить")
                    }
                }
                "plug" -> Button(onClick = { send(if (device?.on == true) "off" else "on") }) {
                    Text(if (device?.on == true) "Выключить" else "Включить")
                }
                "thermostat" -> Row(verticalAlignment = Alignment.CenterVertically) {
                    OutlinedTextField(target, { target = it }, label = { Text("°C") }, modifier = Modifier.width(100.dp))
                    Spacer(Modifier.width(8.dp))
                    Button(onClick = { target.toDoubleOrNull()?.let { send("set_target", it) } }) { Text("Применить") }
                }
            }

            Spacer(Modifier.height(16.dp))
            Text("Телеметрия", style = MaterialTheme.typography.titleMedium)
            Text(
                "t ${latest?.temp ?: "—"}° · h ${latest?.humidity ?: "—"}% · заряд ${latest?.battery ?: "—"}%",
                color = Color.Gray
            )
            Spacer(Modifier.height(8.dp))

            if (temps.size < 2) {
                Box(Modifier.fillMaxWidth().height(180.dp), contentAlignment = Alignment.Center) {
                    Text("ждём данные...", color = Color.Gray)
                }
            } else {
                LineChart(temps, Modifier.fillMaxWidth().height(180.dp))
            }
        }
    }
}

@Composable
private fun LineChart(points: List<Double>, modifier: Modifier = Modifier) {
    val color = MaterialTheme.colorScheme.primary
    Canvas(modifier) {
        if (points.size < 2) return@Canvas
        val max = points.maxOrNull() ?: 1.0
        val min = points.minOrNull() ?: 0.0
        val range = (max - min).coerceAtLeast(1e-6)
        val stepX = size.width / (points.size - 1)

        val path = Path()
        points.forEachIndexed { i, v ->
            val x = i * stepX
            val y = size.height - ((v - min) / range) * size.height
            if (i == 0) path.moveTo(x, y.toFloat()) else path.lineTo(x, y.toFloat())
        }
        drawPath(path, color = color, style = Stroke(width = 6f))
    }
}

private fun typeLabel(type: String) = when (type) {
    "light" -> "Лампа"
    "plug" -> "Розетка"
    "thermostat" -> "Термостат"
    "sensor" -> "Датчик"
    else -> type
}
