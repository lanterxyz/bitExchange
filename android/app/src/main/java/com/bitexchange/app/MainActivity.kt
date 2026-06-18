package com.bitexchange.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import io.ktor.client.*
import io.ktor.client.engine.okhttp.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

// ── Data models ──────────────────────────────────────────

@Serializable
data class Peer(
    val device_id: String,
    val device_name: String,
    val fingerprint: String,
    val path: String = "",
    val rate: Double = 0.0,
    val status: String = "offline",
)

@Serializable
data class TaskInfo(
    val id: String,
    val kind: String = "",
    val target: String = "",
    val payload: String = "",
    val status: String = "pending",
    val progress: Double = 0.0,
    val rate: Double = 0.0,
    val path: String = "",
)

@Serializable
data class ChatLine(
    val type: String,
    val from: String,
    val body: String? = null,
    val filename: String? = null,
)

// ── ViewModel ────────────────────────────────────────────

class BitExchangeViewModel : ViewModel() {
    var apiBase by mutableStateOf("")
    var connected by mutableStateOf(false)
    var peers by mutableStateOf<List<Peer>>(emptyList())
    var tasks by mutableStateOf<List<TaskInfo>>(emptyList())
    var chatLines by mutableStateOf<List<Pair<ChatLine, Boolean>>>(emptyList())
    var selectedTargets by mutableStateOf<List<String>>(emptyList())
    var encrypted by mutableStateOf(false)

    private val client = HttpClient(OkHttp) {
        install(ContentNegotiation) {
            json(Json { ignoreUnknownKeys = true })
        }
    }

    fun connect(url: String) {
        apiBase = url
        viewModelScope.launch {
            try {
                client.get("$apiBase/api/peers")
                connected = true
                startPolling()
            } catch (_: Exception) {
                connected = false
            }
        }
    }

    private fun startPolling() {
        viewModelScope.launch {
            while (connected) {
                try {
                    val resp = client.get("$apiBase/api/peers")
                    peers = Json.decodeFromString(resp.bodyAsText())
                } catch (_: Exception) {}
                try {
                    val resp = client.get("$apiBase/api/tasks")
                    val wrapper = Json.decodeFromString<Map<String, List<TaskInfo>>>(resp.bodyAsText())
                    tasks = wrapper["tasks"] ?: emptyList()
                } catch (_: Exception) {}
                delay(2000)
            }
        }
    }

    fun toggleTarget(deviceId: String) {
        selectedTargets = if (deviceId in selectedTargets) {
            selectedTargets - deviceId
        } else {
            selectedTargets + deviceId
        }
    }

    fun sendText(text: String) {
        if (selectedTargets.isEmpty()) return
        viewModelScope.launch {
            try {
                val body = """{"to_device_ids":${selectedTargets.map { "\"$it\"" }},"body":"$text","encrypted":$encrypted}"""
                client.post("$apiBase/api/send/text") {
                    contentType(ContentType.Application.Json)
                    setBody(body)
                }
                chatLines = chatLines + (ChatLine("text", "me", body = text) to true)
            } catch (_: Exception) {}
        }
    }
}

// ── UI ────────────────────────────────────────────────────

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme(colorScheme = darkColorScheme()) {
                Surface(modifier = Modifier.fillMaxSize(), color = Color(0xFF0f0c29)) {
                    App()
                }
            }
        }
    }
}

private val BgGradient = Brush.verticalGradient(listOf(Color(0xFF0f0c29), Color(0xFF302b63)))
private val AccentPurple = Color(0xFFa78bfa)
private val PanelBg = Color.White.copy(alpha = 0.06f)
private val PanelBorder = Color.White.copy(alpha = 0.12f)

@Composable
fun App(vm: BitExchangeViewModel = viewModel()) {
    if (!vm.connected) {
        ConnectScreen(onConnect = { vm.connect(it) })
    } else {
        MainScreen(vm)
    }
}

@Composable
fun ConnectScreen(onConnect: (String) -> Unit) {
    var url by remember { mutableStateOf("http://127.0.0.1:8080") }
    Column(
        modifier = Modifier.fillMaxSize().background(BgGradient).padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("bitExchange", color = AccentPurple, fontSize = 24.sp, fontWeight = FontWeight.Bold)
        Spacer(modifier = Modifier.height(12.dp))
        Text("连接到 bitexchange-core 实例", color = Color.White.copy(alpha = 0.6f), fontSize = 13.sp)
        Spacer(modifier = Modifier.height(24.dp))
        OutlinedTextField(
            value = url,
            onValueChange = { url = it },
            label = { Text("API 地址") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(modifier = Modifier.height(12.dp))
        Button(onClick = { onConnect(url) }, modifier = Modifier.fillMaxWidth()) {
            Text("连接 →")
        }
    }
}

@Composable
fun MainScreen(vm: BitExchangeViewModel) {
    var inputText by remember { mutableStateOf("") }

    Column(modifier = Modifier.fillMaxSize().background(BgGradient)) {
        // Top: device list
        Column(modifier = Modifier.weight(0.3f).padding(14.dp)) {
            Text("在线设备", color = AccentPurple, fontSize = 11.sp, letterSpacing = 1.sp)
            Spacer(modifier = Modifier.height(8.dp))
            if (vm.peers.isEmpty()) {
                Text("暂无在线设备", color = Color.White.copy(alpha = 0.4f), fontSize = 13.sp)
            }
            LazyColumn {
                items(vm.peers) { peer ->
                    val selected = peer.device_id in vm.selectedTargets
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(10.dp))
                            .background(if (selected) AccentPurple.copy(alpha = 0.12f) else PanelBg)
                            .clickable { vm.toggleTarget(peer.device_id) }
                            .padding(10.dp)
                    ) {
                        Text(peer.device_name, color = Color.White, fontWeight = FontWeight.SemiBold, fontSize = 13.sp)
                        Text("${peer.path} · ${"%.1f".format(peer.rate)}MB/s", color = Color.White.copy(alpha = 0.6f), fontSize = 11.sp)
                    }
                    Spacer(modifier = Modifier.height(6.dp))
                }
            }
        }

        // Middle: chat
        LazyColumn(modifier = Modifier.weight(0.5f).padding(14.dp)) {
            items(vm.chatLines) { (line, outgoing) ->
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = if (outgoing) Arrangement.End else Arrangement.Start,
                ) {
                    Box(
                        modifier = Modifier
                            .clip(RoundedCornerShape(14.dp, 14.dp, if (outgoing) 4.dp else 14.dp, if (outgoing) 14.dp else 4.dp))
                            .background(if (outgoing) Brush.linearGradient(listOf(Color(0xFF7c3aed), AccentPurple)) else Brush.linearGradient(listOf(PanelBg, PanelBg)))
                            .padding(10.dp, 14.dp)
                    ) {
                        Text(line.body ?: line.filename ?: "", color = Color.White, fontSize = 13.sp)
                    }
                }
                Spacer(modifier = Modifier.height(8.dp))
            }
        }

        // Bottom: input
        Row(
            modifier = Modifier.fillMaxWidth().background(PanelBg).padding(14.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            OutlinedTextField(
                value = inputText,
                onValueChange = { inputText = it },
                placeholder = { Text("输入消息...", color = Color.White.copy(alpha = 0.4f)) },
                singleLine = true,
                modifier = Modifier.weight(1f),
            )
            Spacer(modifier = Modifier.width(8.dp))
            Button(onClick = {
                if (inputText.isNotBlank()) {
                    vm.sendText(inputText)
                    inputText = ""
                }
            }) {
                Text("发送 →")
            }
        }
    }
}
