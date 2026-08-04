package cc.orbexa.ylven

import androidx.compose.ui.test.assertExists
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.junit4.StateRestorationTester
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithTag
import androidx.compose.ui.test.performClick
import cc.orbexa.ylven.ui.YlvenApp
import cc.orbexa.ylven.ui.theme.YlvenTheme
import org.junit.Rule
import org.junit.Test

class AppShellTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun shellExposesContractTagsAndEveryTabIsReachable() {
        composeRule.setContent { YlvenTheme { YlvenApp() } }

        composeRule.onNodeWithTag("YL-A-001-C-P00_003-01").assertExists()
        composeRule.onNodeWithTag("YL-A-002-C-P00_004-01").assertExists()
        composeRule.onNodeWithTag("YL-A-003-C-P00_005-01").assertExists()

        listOf("home", "work", "discover", "mine").forEach { tab ->
            composeRule.onNodeWithTag("main-tab-$tab").performClick().assertIsSelected()
            composeRule.onNodeWithTag("tab-content-$tab").assertExists()
        }
    }

    @Test
    fun selectedTabSurvivesStateRestoration() {
        val restorationTester = StateRestorationTester(composeRule)
        restorationTester.setContent { YlvenTheme { YlvenApp() } }

        composeRule.onNodeWithTag("main-tab-discover").performClick().assertIsSelected()
        restorationTester.emulateSavedInstanceStateRestore()

        composeRule.onNodeWithTag("main-tab-discover").assertIsSelected()
        composeRule.onNodeWithTag("tab-content-discover").assertExists()
    }
}
