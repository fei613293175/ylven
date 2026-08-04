#!/usr/bin/env python3
"""Regenerate all derivative YLVEN contracts from contracts/feature-map.yaml.

The YAML feature map is the editable source of truth. This tool deliberately does not
change Feature IDs or implementation status. It regenerates human-readable and machine-
readable indexes so Codex cannot update one layer while leaving another stale.
"""
from __future__ import annotations

import csv
import datetime as dt
import hashlib
import json
import re
from collections import defaultdict
from pathlib import Path
from typing import Any

import yaml

ROOT = Path(__file__).resolve().parents[1]
CONTRACTS = ROOT / "contracts"
HTTP_METHODS = {"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD", "TRACE"}
CSV_FIELDS = [
    "phase", "feature_id", "name", "surface", "screen_or_menu", "user_action",
    "ui_states", "http_method", "endpoint_or_event", "backend_service",
    "data_entities", "admin_control", "permission", "billing_rule", "async_job",
    "acceptance_criteria", "test_ids", "implementation_notes", "status",
]
API_FIELDS = [
    "phase", "feature_id", "surface", "http_method", "endpoint_or_event",
    "backend_service", "permission", "billing_rule", "async_job", "data_entities",
    "admin_control",
]


def now_sgt() -> str:
    tz = dt.timezone(dt.timedelta(hours=8))
    return dt.datetime.now(tz).isoformat(timespec="seconds")


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def load_yaml(path: Path) -> dict[str, Any]:
    return yaml.safe_load(path.read_text(encoding="utf-8")) or {}


def pipe_list(value: Any) -> list[str]:
    return [part.strip() for part in str(value or "").split("|") if part.strip()]


def meaningful(value: str) -> bool:
    return value.strip().lower() not in {"", "none", "n/a", "—", "-"}


def write_csv(path: Path, fieldnames: list[str], rows: list[dict[str, Any]]) -> None:
    with path.open("w", encoding="utf-8-sig", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames, extrasaction="ignore")
        writer.writeheader()
        writer.writerows(rows)


def load_features() -> tuple[dict[str, Any], list[dict[str, Any]]]:
    source = yaml.safe_load((CONTRACTS / "feature-map.yaml").read_text(encoding="utf-8")) or {}
    features = source.get("features") or []
    if not isinstance(features, list) or not features:
        raise ValueError("contracts/feature-map.yaml has no features")
    features = sorted(features, key=lambda item: item["feature_id"])
    return source, features


def regenerate_feature_csv(features: list[dict[str, Any]]) -> None:
    write_csv(CONTRACTS / "feature-map.csv", CSV_FIELDS, features)


def regenerate_api_inventory(features: list[dict[str, Any]]) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    for feature in features:
        endpoint = str(feature.get("endpoint_or_event", ""))
        if not endpoint.startswith("/"):
            continue
        row = {field: feature.get(field, "") for field in API_FIELDS}
        rows.append(row)
    write_csv(CONTRACTS / "api-inventory.csv", API_FIELDS, rows)
    return rows


def regenerate_entity_catalog(features: list[dict[str, Any]]) -> list[dict[str, Any]]:
    catalog: dict[str, dict[str, set[str]]] = defaultdict(
        lambda: {"phases": set(), "services": set(), "feature_ids": set()}
    )
    for feature in features:
        for entity in pipe_list(feature.get("data_entities")):
            if not meaningful(entity):
                continue
            item = catalog[entity]
            item["phases"].add(str(feature["phase"]))
            item["services"].add(str(feature["backend_service"]))
            item["feature_ids"].add(str(feature["feature_id"]))
    entities = [
        {
            "entity": name,
            "phases": sorted(values["phases"]),
            "services": sorted(values["services"]),
            "feature_ids": sorted(values["feature_ids"]),
        }
        for name, values in sorted(catalog.items())
    ]
    output = {"version": "1.4.0", "generated_at": "SOURCE_DERIVED", "entities": entities}
    (CONTRACTS / "database-entity-catalog.yaml").write_text(
        yaml.safe_dump(output, allow_unicode=True, sort_keys=False, width=140), encoding="utf-8"
    )
    return entities


def regenerate_admin_menu_map(features: list[dict[str, Any]]) -> list[str]:
    menus = sorted(
        {
            str(feature.get("admin_control", "")).strip()
            for feature in features
            if meaningful(str(feature.get("admin_control", "")))
        }
    )
    output = {"version": "1.4.0", "generated_at": "SOURCE_DERIVED", "menus": menus}
    (CONTRACTS / "admin-menu-map.yaml").write_text(
        yaml.safe_dump(output, allow_unicode=True, sort_keys=False, width=140), encoding="utf-8"
    )
    return menus


def test_level(test_id: str) -> str:
    suffix = test_id.rsplit("-", 1)[-1].upper()
    return {
        "UNIT": "unit",
        "API": "contract_or_integration",
        "E2E": "ui_or_end_to_end",
        "SEC": "security",
        "PERF": "performance",
    }.get(suffix, "specified")


def regenerate_test_catalog(features: list[dict[str, Any]]) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    for feature in features:
        for test_id in pipe_list(feature.get("test_ids")):
            rows.append(
                {
                    "phase": str(feature["phase"]),
                    "feature_id": str(feature["feature_id"]),
                    "test_id": test_id,
                    "level": test_level(test_id),
                    "required_result": (
                        "PASS or documented BLOCKED_EXTERNAL with evidence; release cannot leave PLANNED"
                    ),
                }
            )
    write_csv(
        CONTRACTS / "test-catalog.csv",
        ["phase", "feature_id", "test_id", "level", "required_result"],
        rows,
    )
    return rows


def operation_id(method: str, path: str, first_feature: str) -> str:
    slug = re.sub(r"[^a-zA-Z0-9]+", "_", path.strip("/")).strip("_").lower()
    return f"{first_feature.lower().replace('-', '_')}_{method.lower()}_{slug}"[:180]


def clean_list(values: list[str]) -> list[str]:
    return sorted({value.strip() for value in values if meaningful(value)})


def regenerate_openapi(
    features: list[dict[str, Any]], api_rows: list[dict[str, Any]]
) -> tuple[int, int]:
    feature_by_id = {feature["feature_id"]: feature for feature in features}
    groups: dict[tuple[str, str], list[dict[str, Any]]] = defaultdict(list)
    mapping_count = 0
    for row in api_rows:
        methods = [method.upper() for method in pipe_list(row["http_method"])]
        for method in methods:
            if method not in HTTP_METHODS:
                raise ValueError(f"Unsupported HTTP method {method!r} for {row['feature_id']}")
            groups[(method, row["endpoint_or_event"])].append(row)
            mapping_count += 1

    paths: dict[str, dict[str, Any]] = {}
    for (method, path), operation_rows in sorted(groups.items(), key=lambda item: (item[0][1], item[0][0])):
        feature_ids = sorted({row["feature_id"] for row in operation_rows})
        feature_items = [feature_by_id[feature_id] for feature_id in feature_ids]
        services = clean_list([str(row["backend_service"]) for row in operation_rows])
        surfaces = clean_list([str(row["surface"]) for row in operation_rows])
        operation: dict[str, Any] = {
            "operationId": operation_id(method, path, feature_ids[0]),
            "summary": " / ".join(str(item["name"]) for item in feature_items),
            "tags": clean_list(surfaces + services)[:8],
            "x-feature-id": feature_ids[0],
            "x-feature-ids": feature_ids,
            "x-phases": clean_list([str(row["phase"]) for row in operation_rows]),
            "x-services": services,
            "x-permissions": clean_list([str(row["permission"]) for row in operation_rows]),
            "x-billing-rules": clean_list([str(row["billing_rule"]) for row in operation_rows]) or ["none"],
            "x-async-jobs": clean_list([str(row["async_job"]) for row in operation_rows]) or ["none"],
            "responses": {
                "200": {"description": "Success"},
                "400": {"$ref": "#/components/responses/BadRequest"},
                "401": {"$ref": "#/components/responses/Unauthorized"},
                "403": {"$ref": "#/components/responses/Forbidden"},
                "409": {"$ref": "#/components/responses/Conflict"},
                "429": {"$ref": "#/components/responses/RateLimited"},
                "500": {"$ref": "#/components/responses/InternalError"},
            },
        }
        paths.setdefault(path, {})[method.lower()] = operation

    error_content = {"application/json": {"schema": {"$ref": "#/components/schemas/ErrorEnvelope"}}}
    spec = {
        "openapi": "3.1.0",
        "info": {
            "title": "YLVEN API Contract Skeleton",
            "version": "0.1.0",
            "description": (
                "Generated from feature-map.yaml. Each implementation phase must replace generic schemas with "
                "concrete request/response contracts while preserving x-feature-ids traceability."
            ),
        },
        "servers": [
            {"url": "https://api.orbexa.cc"},
            {"url": "https://gateway.orbexa.cc"},
        ],
        "paths": paths,
        "components": {
            "securitySchemes": {
                "BearerAuth": {"type": "http", "scheme": "bearer", "bearerFormat": "JWT"},
                "YlvenApiKey": {"type": "apiKey", "in": "header", "name": "Authorization"},
            },
            "schemas": {
                "ErrorEnvelope": {
                    "type": "object",
                    "required": ["error"],
                    "properties": {
                        "error": {
                            "type": "object",
                            "required": ["code", "message", "request_id", "retryable"],
                            "properties": {
                                "code": {"type": "string"},
                                "message": {"type": "string"},
                                "request_id": {"type": "string"},
                                "retryable": {"type": "boolean"},
                                "details": {"type": "object", "additionalProperties": True},
                            },
                        }
                    },
                }
            },
            "responses": {
                "BadRequest": {"description": "Invalid request", "content": error_content},
                "Unauthorized": {"description": "Authentication required", "content": error_content},
                "Forbidden": {"description": "Permission denied", "content": error_content},
                "Conflict": {"description": "State or idempotency conflict", "content": error_content},
                "RateLimited": {"description": "Rate limit exceeded", "content": error_content},
                "InternalError": {"description": "Internal error", "content": error_content},
            },
        },
    }
    (CONTRACTS / "openapi-skeleton.yaml").write_text(
        yaml.safe_dump(spec, allow_unicode=True, sort_keys=False, width=140), encoding="utf-8"
    )
    return len(groups), mapping_count


def regenerate_feature_schema() -> None:
    schema = {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$id": "https://docs.orbexa.cc/contracts/feature-map.schema.json",
        "title": "YLVEN Feature Map",
        "type": "object",
        "required": ["version", "features"],
        "properties": {
            "version": {"type": "string"},
            "features": {
                "type": "array",
                "minItems": 1,
                "items": {
                    "type": "object",
                    "required": CSV_FIELDS,
                    "properties": {
                        "phase": {"type": "string", "pattern": "^P(0[0-9]|1[0-3])$"},
                        "feature_id": {"type": "string", "pattern": "^P(0[0-9]|1[0-3])-[0-9]{3}$"},
                        **{field: {"type": "string"} for field in CSV_FIELDS if field not in {"phase", "feature_id"}},
                    },
                    "additionalProperties": True,
                },
            },
        },
        "additionalProperties": False,
    }
    (CONTRACTS / "feature-map.schema.json").write_text(
        json.dumps(schema, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )


def regenerate_feature_matrix(features: list[dict[str, Any]], api_row_count: int) -> None:
    lines = [
        "# YLVEN 前端—后端—后台—数据—测试功能追踪矩阵",
        "",
        f"> 版本：1.4.0；总功能数：**{len(features)}**；HTTP/API 功能数：**{api_row_count}**。",
        "",
        "本文件是可读索引；机器校验使用 `feature-map.yaml`、`feature-map.schema.json`、`feature-map.csv`、`api-inventory.csv` 与 `test-catalog.csv`。任何前端功能、后台菜单、收费能力或异步任务都必须具有 Feature ID。",
        "",
        "## 字段解释",
        "",
        "- **Surface**：Android、管理后台、开发者门户、公共 API、后端、Worker 或运维入口。",
        "- **接口/事件**：实际 HTTP 契约、领域事件、脚本或客户端内部合同。",
        "- **数据**：该能力直接读取或写入的权威实体。",
        "- **后台控制**：运营、配置、诊断或审计入口；“—”表示仅由固定工程规则控制。",
        "- **计费**：`usage_metered` 表示必须经过额度预检并生成价格快照和用量事件。",
        "- **验收**：除正常路径外，还必须验证权限、重复提交、弱网、超时、幂等和可恢复错误。",
        "",
    ]
    by_phase: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for feature in features:
        by_phase[str(feature["phase"])].append(feature)
    for phase in sorted(by_phase):
        lines.extend(
            [
                f"## {phase}",
                "",
                "| Feature ID | 功能 | Surface / 页面 | 方法与接口/事件 | 后端模块 | 数据实体 | 后台控制 | 计费/任务 |",
                "|---|---|---|---|---|---|---|---|",
            ]
        )
        for feature in by_phase[phase]:
            cells = [
                f"`{feature['feature_id']}`",
                str(feature["name"]),
                f"{feature['surface']} / {feature['screen_or_menu']}",
                f"`{feature['http_method']} {feature['endpoint_or_event']}`",
                str(feature["backend_service"]),
                str(feature["data_entities"]).replace("|", "、"),
                str(feature["admin_control"]),
                f"{feature['billing_rule']} / {feature['async_job']}",
            ]
            lines.append("| " + " | ".join(cell.replace("|", "/") for cell in cells) + " |")
        lines.extend(
            [
                "",
                f"### {phase} 统一验收要求",
                "",
                "1. 本阶段全部 Feature ID 必须能在代码、API、数据库迁移、后台菜单或自动化测试中追踪，不能只修改文档状态。",
                "2. Android/网页端按适用范围实现 loading、empty、success、validation error、unauthorized、rate-limited、offline/timeout 和 retry 状态。",
                "3. 接口遵循统一错误合同、请求 ID、幂等、权限和审计要求；收费能力保存价格快照并可对账。",
                "4. 发布脚本必须在合同、相关测试、真实构建或部署证据缺失时失败。",
                "",
            ]
        )
    (CONTRACTS / "FEATURE_MATRIX.md").write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")


def regenerate_manifest(
    feature_count: int,
    api_row_count: int,
    openapi_operation_count: int,
    api_feature_mapping_count: int,
    entity_count: int,
    test_count: int,
    admin_menu_count: int,
) -> None:
    files = []
    for path in sorted(CONTRACTS.iterdir()):
        if not path.is_file() or path.name in {"CONTRACT_MANIFEST.json", "CURRENT_PHASE.yaml"}:
            continue
        files.append({"path": path.name, "size": path.stat().st_size, "sha256": sha256(path)})
    manifest = {
        "schema_version": "1.4.0",
        "generated_at": "SOURCE_DERIVED",
        "source_of_truth": "feature-map.yaml",
        "feature_count": feature_count,
        "api_inventory_row_count": api_row_count,
        "api_feature_mapping_count": api_feature_mapping_count,
        "openapi_operation_count": openapi_operation_count,
        "database_entity_count": entity_count,
        "admin_menu_count": admin_menu_count,
        "test_count": test_count,
        "ui_page_count": len((load_yaml(CONTRACTS / "ui-page-catalog.yaml") or {}).get("pages") or []),
        "ui_state_count": sum(1 for _ in csv.DictReader((CONTRACTS / "ui-state-catalog.csv").open(encoding="utf-8-sig", newline=""))),
        "ui_mockup_count": sum(1 for _ in csv.DictReader((CONTRACTS / "mockup-manifest.csv").open(encoding="utf-8-sig", newline=""))),
        "work_packet_count": len((load_yaml(CONTRACTS / "work-packet-map.yaml") or {}).get("work_packets") or []),
        "files": files,
    }
    (CONTRACTS / "CONTRACT_MANIFEST.json").write_text(
        json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )


def remove_stale_state_mirrors() -> None:
    """Delete deprecated contract-directory runtime mirrors.

    Runtime state has exactly one authority at repository root/status. Older package
    revisions mirrored current state under contracts/, which can diverge after a
    session change. Regeneration therefore removes those stale files instead of
    silently carrying a second state source forward.
    """
    for name in ("CURRENT_PHASE.yaml", "CURRENT_WORK_PACKET.yaml"):
        stale = CONTRACTS / name
        if stale.exists():
            stale.unlink()


def main() -> int:
    remove_stale_state_mirrors()
    _, features = load_features()
    regenerate_feature_csv(features)
    api_rows = regenerate_api_inventory(features)
    entities = regenerate_entity_catalog(features)
    menus = regenerate_admin_menu_map(features)
    tests = regenerate_test_catalog(features)
    regenerate_feature_schema()
    regenerate_feature_matrix(features, len(api_rows))
    operations, mappings = regenerate_openapi(features, api_rows)
    regenerate_manifest(
        feature_count=len(features),
        api_row_count=len(api_rows),
        openapi_operation_count=operations,
        api_feature_mapping_count=mappings,
        entity_count=len(entities),
        test_count=len(tests),
        admin_menu_count=len(menus),
    )
    print(
        "Contracts regenerated: "
        f"{len(features)} features, {len(api_rows)} API inventory rows, "
        f"{operations} OpenAPI operations, {mappings} feature-operation mappings, "
        f"{len(entities)} entities, {len(menus)} admin menus and {len(tests)} tests."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
