package cc.orbexa.ylven

import cc.orbexa.ylven.ui.MainTab
import cc.orbexa.ylven.ui.theme.YlvenDimensions
import org.junit.Assert.assertEquals
import org.junit.Test

class AppStateTest {
    @Test
    fun mainNavigationHasTheContractedOrderAndStableTags() {
        assertEquals(listOf("首页", "工作", "发现", "我的"), MainTab.entries.map { it.label })
        assertEquals(
            listOf("main-tab-home", "main-tab-work", "main-tab-discover", "main-tab-mine"),
            MainTab.entries.map { it.testTag },
        )
    }

    @Test
    fun shellDimensionsMatchYlDs120() {
        assertEquals(56f, YlvenDimensions.TopAppBarHeight.value)
        assertEquals(64f, YlvenDimensions.BottomNavigationHeight.value)
        assertEquals(16f, YlvenDimensions.PageHorizontal.value)
        assertEquals(24f, YlvenDimensions.SectionGap.value)
        assertEquals(24f, YlvenDimensions.Icon.value)
        assertEquals(48f, YlvenDimensions.MinimumTouchTarget.value)
    }
}
