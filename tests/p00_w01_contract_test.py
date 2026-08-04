from copy import deepcopy
from pathlib import Path
import re
import unittest

import yaml


ROOT = Path(__file__).resolve().parents[1]


def load_yaml(relative_path: str) -> dict:
    return yaml.safe_load((ROOT / relative_path).read_text(encoding="utf-8"))


def recursive_merge(base: dict, override: dict) -> dict:
    result = deepcopy(base)
    for key, value in override.items():
        if key == "extends":
            continue
        if isinstance(value, dict) and isinstance(result.get(key), dict):
            result[key] = recursive_merge(result[key], value)
        else:
            result[key] = deepcopy(value)
    return result


def value_at_path(document: dict, dotted_path: str):
    value = document
    for part in dotted_path.split("."):
        value = value[part]
    return value


class P00W01ContractTest(unittest.TestCase):
    def test_repository_module_boundaries_are_unique_and_packet_aligned(self):
        modules = load_yaml("repository/modules.yaml")
        rows = modules["modules"]
        self.assertEqual("YLVEN", modules["repository_name"])
        self.assertEqual(len(rows), len({row["id"] for row in rows}))
        self.assertEqual(len(rows), len({row["path"] for row in rows}))
        self.assertTrue(all(row["owner"] and row["runtime"] for row in rows))

        by_id = {row["id"]: row for row in rows}
        self.assertEqual("active_in_P00-W01", by_id["android-app"]["status"])
        for module_id in ("core-api", "ai-runtime", "developer-gateway", "worker", "scheduler"):
            self.assertEqual("planned_for_P00-W02", by_id[module_id]["status"])
        self.assertTrue((ROOT / by_id["android-app"]["path"] / "src/main/AndroidManifest.xml").is_file())

    def test_every_environment_layer_resolves_against_schema_without_plaintext_secrets(self):
        schema = load_yaml("config/schema.yaml")
        base = load_yaml("config/base.yaml")
        forbidden = [re.compile(pattern) for pattern in schema["forbidden_value_patterns"]]

        self.assertEqual("recursive_map_override", schema["merge_strategy"])
        self.assertEqual(["local", "staging", "production"], schema["environment_layers"])
        for environment in schema["environment_layers"]:
            path = ROOT / f"config/environments/{environment}.yaml"
            layer = yaml.safe_load(path.read_text(encoding="utf-8"))
            self.assertEqual("../base.yaml", layer["extends"])
            self.assertEqual(environment, layer["environment"])
            effective = recursive_merge(base, layer)
            for required_path in schema["required_paths"]:
                self.assertIsNotNone(value_at_path(effective, required_path), required_path)
            self.assertEqual(environment, effective["environment"])

        for path in [ROOT / "config/base.yaml", *sorted((ROOT / "config/environments").glob("*.yaml"))]:
            content = path.read_text(encoding="utf-8")
            for pattern in forbidden:
                self.assertIsNone(pattern.search(content), f"{path.name}: {pattern.pattern}")
        self.assertTrue(all(str(value).startswith(schema["secret_reference_prefix"]) for value in base["secret_refs"].values()))

    def test_android_build_and_ui_sources_match_p00_contracts(self):
        release = load_yaml("contracts/release-version-matrix.yaml")
        p00 = next(row for row in release["phases"] if row["phase"] == "P00")
        gradle = (ROOT / "android/app/build.gradle.kts").read_text(encoding="utf-8")
        settings = (ROOT / "settings.gradle.kts").read_text(encoding="utf-8")
        dimensions = (ROOT / "android/app/src/main/java/cc/orbexa/ylven/ui/theme/Dimensions.kt").read_text(encoding="utf-8")
        colors = (ROOT / "android/app/src/main/java/cc/orbexa/ylven/ui/theme/Color.kt").read_text(encoding="utf-8")

        self.assertRegex(gradle, rf'applicationId\s*=\s*"{re.escape(release["repository"]["application_id"])}"')
        self.assertRegex(gradle, rf'versionName\s*=\s*"{re.escape(p00["version_name"])}"')
        self.assertRegex(gradle, rf'versionCode\s*=\s*{p00["version_code"]}\b')
        self.assertIn('include(":app")', settings)
        for declaration in ("TopAppBarHeight = 56.dp", "BottomNavigationHeight = 64.dp", "PageHorizontal = 16.dp", "SectionGap = 24.dp"):
            self.assertIn(declaration, dimensions)
        for color in ("0xFF5B61F6", "0xFFF6F7FB", "0xFFFFFFFF", "0xFF101828", "0xFF475467", "0xFFE4E7EC"):
            self.assertIn(color, colors)


if __name__ == "__main__":
    unittest.main()
