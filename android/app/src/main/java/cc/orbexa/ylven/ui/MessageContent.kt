package cc.orbexa.ylven.ui

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.unit.sp

/** Small dependency-free Markdown renderer for persisted assistant content.
 * It deliberately renders only safe structural Markdown (headings, bullets,
 * fenced code and pipe tables); links remain text and are never auto-opened.
 */
@Composable
fun MessageContent(body: String, modifier: Modifier = Modifier, onCopyCode: (String) -> Unit = {}) {
    val lines = body.replace("\r\n", "\n").split('\n')
    Column(modifier.testTag("YL-A-026-C-P03_018-01")) {
        var index = 0
        while (index < lines.size) {
            val line = lines[index]
            if (line.trimStart().startsWith("```")) {
                val code = buildString {
                    index++
                    while (index < lines.size && !lines[index].trimStart().startsWith("```")) {
                        append(lines[index]).append('\n'); index++
                    }
                }.trimEnd()
                Row(Modifier.horizontalScroll(rememberScrollState()).testTag("YL-A-027-C-P03_019-01")) {
                    Column {
                        TextButton(
                            onClick = { onCopyCode(code) },
                            modifier = Modifier.testTag("p03-copy-code"),
                        ) { Text("复制代码") }
                        Text(code)
                    }
                }
            } else if (line.contains('|') && index + 1 < lines.size && lines[index + 1].contains("---")) {
                val headers = line.trim().trim('|').split('|').map(String::trim)
                index += 2
                Row(Modifier.horizontalScroll(rememberScrollState()).testTag("YL-A-028-C-P03_020-01")) {
                    headers.forEach { Text("$it   ") }
                }
                while (index < lines.size && lines[index].contains('|')) {
                    Row(Modifier.horizontalScroll(rememberScrollState())) { lines[index].trim().trim('|').split('|').map(String::trim).forEach { Text("$it   ") } }; index++
                }
                continue
            } else {
                Text(line.removePrefix("# ").removePrefix("## "), fontSize = 14.sp, lineHeight = 20.sp)
            }
            index++
        }
    }
}
