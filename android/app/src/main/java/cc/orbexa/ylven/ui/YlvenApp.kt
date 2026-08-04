package cc.orbexa.ylven.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Explore
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.WorkOutline
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import cc.orbexa.ylven.ui.theme.YlvenDimensions

private const val AppShellTag = "YL-A-001-C-P00_003-01"
private const val DesignSystemTag = "YL-A-002-C-P00_004-01"
private const val NavigationTag = "YL-A-003-C-P00_005-01"

enum class MainTab(
    val label: String,
    val icon: androidx.compose.ui.graphics.vector.ImageVector,
    val testTag: String,
) {
    HOME("首页", Icons.Default.Home, "main-tab-home"),
    WORK("工作", Icons.Default.WorkOutline, "main-tab-work"),
    DISCOVER("发现", Icons.Default.Explore, "main-tab-discover"),
    MINE("我的", Icons.Default.Person, "main-tab-mine"),
}

@Composable
@OptIn(ExperimentalMaterial3Api::class)
fun YlvenApp() {
    var selectedTab by rememberSaveable { mutableStateOf(MainTab.HOME.name) }
    val tab = MainTab.valueOf(selectedTab)

    Surface(
        modifier = Modifier.fillMaxSize().testTag(AppShellTag),
        color = MaterialTheme.colorScheme.background,
    ) {
        Scaffold(
            topBar = {
                TopAppBar(
                    modifier = Modifier.height(YlvenDimensions.TopAppBarHeight),
                    title = { Text("YLVEN", style = MaterialTheme.typography.titleMedium) },
                )
            },
            bottomBar = {
                NavigationBar(
                    modifier = Modifier
                        .height(YlvenDimensions.BottomNavigationHeight)
                        .testTag(NavigationTag),
                ) {
                    MainTab.entries.forEach { item ->
                        NavigationBarItem(
                            modifier = Modifier.testTag(item.testTag),
                            selected = item == tab,
                            onClick = { selectedTab = item.name },
                            icon = { Icon(item.icon, contentDescription = item.label, modifier = Modifier.size(YlvenDimensions.Icon)) },
                            label = { Text(item.label) },
                        )
                    }
                }
            },
        ) { innerPadding ->
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding)
                    .padding(horizontal = YlvenDimensions.PageHorizontal, vertical = YlvenDimensions.PageTop),
                verticalArrangement = Arrangement.spacedBy(YlvenDimensions.SectionGap),
            ) {
                Text(tab.label, style = MaterialTheme.typography.headlineSmall)
                TabContent(tab)
                DesignTokenSummary()
            }
        }
    }
}

@Composable
private fun TabContent(tab: MainTab) {
    val description = when (tab) {
        MainTab.HOME -> "AI 对话中心将在这里承载会话与模型选择。"
        MainTab.WORK -> "项目、文件和作品会按工作流集中管理。"
        MainTab.DISCOVER -> "发现内容由 YLVEN 服务端配置并按权限展示。"
        MainTab.MINE -> "账户、安全、设备与偏好设置集中在这里。"
    }
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(YlvenDimensions.CardRadius),
        border = BorderStroke(YlvenDimensions.Border, MaterialTheme.colorScheme.outline),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(YlvenDimensions.CardPadding)
                .testTag("tab-content-${tab.name.lowercase()}"),
            verticalArrangement = Arrangement.spacedBy(YlvenDimensions.InlineGap),
        ) {
            Text("当前暂无内容", style = MaterialTheme.typography.titleMedium)
            Text(description, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun DesignTokenSummary() {
    Card(
        modifier = Modifier.fillMaxWidth().testTag(DesignSystemTag),
        shape = RoundedCornerShape(YlvenDimensions.CardRadius),
        border = BorderStroke(YlvenDimensions.Border, MaterialTheme.colorScheme.outline),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
    ) {
        Column(
            modifier = Modifier.padding(YlvenDimensions.CardPadding),
            verticalArrangement = Arrangement.spacedBy(YlvenDimensions.CardGap),
        ) {
            Text("YLVEN 设计系统", style = MaterialTheme.typography.titleMedium)
            Text("YL-DS-1.2.0", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Row(horizontalArrangement = Arrangement.spacedBy(YlvenDimensions.InlineGap), verticalAlignment = Alignment.CenterVertically) {
                Surface(
                    modifier = Modifier.size(YlvenDimensions.Icon),
                    shape = RoundedCornerShape(YlvenDimensions.InlineGap),
                    color = MaterialTheme.colorScheme.primary,
                ) {}
                Text("主题 Token 已按系统浅色/深色模式应用", style = MaterialTheme.typography.bodyMedium)
            }
        }
    }
}
