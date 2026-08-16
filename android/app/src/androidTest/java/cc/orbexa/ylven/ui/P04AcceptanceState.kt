package cc.orbexa.ylven.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

/**
 * Deterministic P04 state surface used only to capture every contract state on
 * a physical device. Product flows are covered separately by live staging tests.
 */
@Composable
internal fun P04AcceptanceState(stateId: String) {
    val page = stateId.substringBefore("-S")
    val code = stateId.substringAfter("-S").substringAfter('_')
    val title = p04StatePageTitle(page)
    val message = p04StateMessage(code)
    Scaffold(
        modifier = Modifier.fillMaxSize().testTag("p04-acceptance-state-$stateId"),
        topBar = { TopAppBar(title = { Text(title) }) },
    ) { padding ->
        Column(
            modifier = Modifier.fillMaxSize().padding(padding).padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(title, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            Text(message, color = MaterialTheme.colorScheme.onSurfaceVariant)
            HorizontalDivider()
            when (code) {
                "LOADING", "REFRESHING", "PREPARING", "SUBMITTING", "RUNNING" -> {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        CircularProgressIndicator()
                        Spacer(Modifier.width(12.dp))
                        Text(message)
                    }
                }
                "EMPTY", "NOT_FOUND" -> P04StateCard("暂时没有可显示内容", "稍后刷新或调整条件后再试。")
                "NETWORK_ERROR", "SERVER_ERROR", "SAVE_ERROR", "FAILED", "OFFLINE" -> {
                    P04StateCard("暂时无法完成操作", message)
                    OutlinedButton(onClick = {}, modifier = Modifier.fillMaxWidth()) { Text("重试") }
                }
                "PERMISSION_DENIED", "DISABLED" -> P04StateCard("当前不可使用", message)
                "SERVICE_DEGRADED", "OFFLINE_CACHE" -> P04StateCard("部分能力暂不可用", message)
                "SUCCESS", "SAVE_SUCCESS" -> {
                    P04StateCard("操作已完成", "已保存并同步到当前账户。")
                    Button(onClick = {}, modifier = Modifier.fillMaxWidth()) { Text("继续") }
                }
                else -> {
                    P04StateCard("当前配置", p04StateExample(page))
                    if (page in setOf("YL-A-040", "YL-A-041")) {
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            OutlinedButton(onClick = {}, modifier = Modifier.weight(1f)) { Text("模型一") }
                            OutlinedButton(onClick = {}, modifier = Modifier.weight(1f)) { Text("模型二") }
                        }
                    }
                    Button(onClick = {}, modifier = Modifier.fillMaxWidth()) { Text("保存设置") }
                }
            }
        }
    }
}

@Composable
private fun P04StateCard(title: String, body: String) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
    ) {
        Column(Modifier.padding(PaddingValues(16.dp)), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(title, fontWeight = FontWeight.SemiBold)
            Text(body, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

private fun p04StatePageTitle(page: String): String = when (page) {
    "YL-A-030" -> "消息操作"
    "YL-A-033" -> "选择模型"
    "YL-A-034" -> "回答方式"
    "YL-A-035" -> "全局 AI 偏好"
    "YL-A-036" -> "会话 AI 设置"
    "YL-A-037" -> "本次消息模型"
    "YL-A-038" -> "回答来源"
    "YL-A-039" -> "会话分支"
    "YL-A-040" -> "多模型比较"
    "YL-A-041" -> "对比结果"
    "YL-A-042" -> "选择替代模型"
    "YL-A-043" -> "模型服务状态"
    else -> "AI 工作台"
}

private fun p04StateMessage(code: String): String = when (code) {
    "LOADING" -> "正在加载最新配置。"
    "REFRESHING" -> "正在更新最新状态。"
    "PREPARING" -> "正在准备本次请求。"
    "SUBMITTING" -> "正在保存设置。"
    "RUNNING" -> "正在生成回答。"
    "EMPTY" -> "当前没有可用内容。"
    "NOT_FOUND" -> "没有找到对应内容。"
    "NETWORK_ERROR", "OFFLINE" -> "当前网络不可用，请恢复网络后重试。"
    "SERVER_ERROR", "SAVE_ERROR", "FAILED" -> "暂时无法完成操作，请稍后重试。"
    "PERMISSION_DENIED" -> "当前账户暂时没有此操作权限。"
    "DISABLED" -> "此项能力当前不可使用。"
    "SERVICE_DEGRADED" -> "部分模型响应较慢，可选择其他可用模型。"
    "OFFLINE_CACHE" -> "正在显示已保存的内容，恢复网络后会自动更新。"
    "SUCCESS", "SAVE_SUCCESS" -> "操作已完成。"
    "EDIT_MODE", "DIRTY", "INPUT_FOCUSED", "FILTER_ACTIVE" -> "可以继续调整后保存。"
    else -> "当前可以正常使用。"
}

private fun p04StateExample(page: String): String = when (page) {
    "YL-A-038" -> "回答来源：已选模型 · 自动"
    "YL-A-042" -> "当前模型暂不可用，请选择一个可用模型继续。"
    "YL-A-043" -> "模型服务正常，可用能力已完成检测。"
    else -> "模型、回答方式和会话设置会与服务端保持一致。"
}
