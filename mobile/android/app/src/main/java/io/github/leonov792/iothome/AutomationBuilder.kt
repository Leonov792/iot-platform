package io.github.leonov792.iothome

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.google.gson.Gson
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

// Конструктор автоматизаций: простые блоки «Если / То», сохранение в automation-сервис.

private data class RuleDraft(
    var name: String = "",
    var deviceId: String = "",
    var field: String = "",
    var op: String = "gt",
    var threshold: Double = 0.0,
    var actionDeviceId: String = "",
    var relay: String = "",
    var value: Boolean = true
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AutomationBuilderScreen(
    automationUrl: String = "http://10.0.2.2:8091",
    automationToken: String = "dev-rules-token"
) {
    val scope = rememberCoroutineScope()
    var rules by remember { mutableStateOf(listOf(RuleDraft())) }
    var message by remember { mutableStateOf<String?>(null) }

    fun save() {
        val payload = rules
            .filter { it.deviceId.isNotBlank() && it.field.isNotBlank() }
            .map { r ->
                mapOf(
                    "id" to (if (r.name.isBlank()) r.deviceId else r.name),
                    "name" to r.name,
                    "condition" to mapOf(
                        "device_id" to r.deviceId,
                        "field" to r.field,
                        "op" to r.op,
                        "value" to r.threshold
                    ),
                    "actions" to listOf(
                        mapOf(
                            "type" to "modbus_write",
                            "device_id" to r.actionDeviceId,
                            "relay" to r.relay,
                            "value" to r.value
                        )
                    ),
                    "cooldown" to "1m"
                )
            }

        scope.launch {
            val ok = withContext(Dispatchers.IO) {
                runCatching {
                    val body = Gson().toJson(payload).toRequestBody("application/json".toMediaType())
                    val req = Request.Builder()
                        .url(automationUrl + "/v1/rules")
                        .header("X-Automation-Token", automationToken)
                        .put(body)
                        .build()
                    OkHttpClient().newCall(req).execute().use { it.isSuccessful }
                }.getOrDefault(false)
            }
            message = if (ok) "Сохранено" else "Ошибка сохранения"
        }
    }

    Scaffold(
        topBar = { TopAppBar(title = { Text("Автоматизации") }) }
    ) { pad ->
        Column(Modifier.fillMaxSize().padding(pad).padding(16.dp)) {
            LazyColumn(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                items(rules.size) { i ->
                    RuleCard(
                        rule = rules[i],
                        onChange = { rules = rules.toMutableList().also { it[i] = it[i] } }
                    )
                }
                item {
                    Button(
                        onClick = { rules = rules + RuleDraft() },
                        modifier = Modifier.fillMaxWidth()
                    ) { Text("+ Добавить правило") }
                }
            }
            Button(onClick = { save() }, modifier = Modifier.fillMaxWidth().padding(top = 8.dp)) {
                Text("Сохранить")
            }
            message?.let { Text(it) }
        }
    }
}

@Composable
private fun RuleCard(rule: RuleDraft, onChange: () -> Unit) {
    Card(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            OutlinedTextField(
                rule.name,
                { rule.name = it; onChange() },
                label = { Text("Название") },
                modifier = Modifier.fillMaxWidth()
            )
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                Text("ЕСЛИ")
                OutlinedTextField(rule.deviceId, { rule.deviceId = it; onChange() }, label = { Text("device") }, modifier = Modifier.weight(1f))
                OutlinedTextField(rule.field, { rule.field = it; onChange() }, label = { Text("поле") }, modifier = Modifier.weight(1f))
            }
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                OutlinedTextField(rule.op, { rule.op = it; onChange() }, label = { Text("op") }, modifier = Modifier.weight(1f))
                OutlinedTextField(
                    rule.threshold.toString(),
                    { v -> rule.threshold = v.toDoubleOrNull() ?: 0.0; onChange() },
                    label = { Text("значение") },
                    modifier = Modifier.weight(1f)
                )
            }
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                Text("ТО")
                OutlinedTextField(rule.actionDeviceId, { rule.actionDeviceId = it; onChange() }, label = { Text("device") }, modifier = Modifier.weight(1f))
                OutlinedTextField(rule.relay, { rule.relay = it; onChange() }, label = { Text("реле") }, modifier = Modifier.weight(1f))
                Switch(checked = rule.value, onCheckedChange = { rule.value = it; onChange() })
            }
        }
    }
}
