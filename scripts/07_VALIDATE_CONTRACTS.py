#!/usr/bin/env python3
from __future__ import annotations
import argparse, csv, json, re, sys
from pathlib import Path
try:
    import yaml
except Exception as exc:
    print(f'PyYAML is required: {exc}', file=sys.stderr); sys.exit(2)

ROOT = Path(__file__).resolve().parents[1]
PHASES = [f'P{i:02d}' for i in range(14)]
FINAL_STATUSES = {'IMPLEMENTED', 'DEFERRED_WITH_REASON', 'BLOCKED_EXTERNAL'}
HTTP_METHODS = {'GET','POST','PUT','PATCH','DELETE','OPTIONS','HEAD','TRACE'}

def load_yaml(path: Path): return yaml.safe_load(path.read_text(encoding='utf-8'))
def split_pipe(value): return [x.strip() for x in str(value or '').split('|') if x.strip()]

def main() -> int:
    ap=argparse.ArgumentParser(); ap.add_argument('--phase'); ap.add_argument('--release',action='store_true'); args=ap.parse_args()
    errors=[]
    for p in sorted((ROOT/'contracts').glob('*.yaml')):
        try: load_yaml(p)
        except Exception as exc: errors.append(f'invalid YAML {p.relative_to(ROOT)}: {exc}')
    for p in sorted((ROOT/'contracts').glob('*.json')):
        try: json.loads(p.read_text(encoding='utf-8'))
        except Exception as exc: errors.append(f'invalid JSON {p.relative_to(ROOT)}: {exc}')

    fmap=ROOT/'contracts/feature-map.yaml'
    try: data=load_yaml(fmap) or {}
    except Exception as exc: print(f'ERROR: cannot parse {fmap}: {exc}',file=sys.stderr); return 1
    feats=data.get('features') or []; ids=[str(f.get('feature_id','')) for f in feats]; id_set=set(ids)
    if len(ids)!=len(id_set): errors.append('duplicate feature_id values')
    if not feats: errors.append('feature map is empty')
    required_fields=['phase','feature_id','name','surface','screen_or_menu','user_action','ui_states','endpoint_or_event','backend_service','data_entities','admin_control','permission','billing_rule','async_job','acceptance_criteria','test_ids']
    for f in feats:
        fid=str(f.get('feature_id',''))
        for key in required_fields:
            if not str(f.get(key,'')).strip(): errors.append(f'{fid}: missing {key}')
        if not re.fullmatch(r'P\d{2}-\d{3}',fid): errors.append(f'invalid feature id {fid}')
        if f.get('phase') != fid[:3]: errors.append(f'{fid}: phase mismatch')
        if f.get('phase') not in PHASES: errors.append(f'{fid}: unknown phase {f.get("phase")}')
        if len(str(f.get('acceptance_criteria','')).strip()) < 40: errors.append(f'{fid}: acceptance criteria is too brief')
        methods=split_pipe(f.get('http_method'))
        endpoint=str(f.get('endpoint_or_event',''))
        if endpoint.startswith('/'):
            for method in methods:
                if method.upper() not in HTTP_METHODS: errors.append(f'{fid}: unsupported HTTP method {method}')

    try:
        with (ROOT/'contracts/feature-map.csv').open(encoding='utf-8-sig',newline='') as fh: csv_rows=list(csv.DictReader(fh))
        csv_ids=[r.get('feature_id','') for r in csv_rows]
        if len(csv_rows)!=len(feats): errors.append(f'CSV/YAML feature count mismatch {len(csv_rows)} != {len(feats)}')
        if set(csv_ids)!=id_set: errors.append('CSV/YAML feature IDs differ')
        if len(csv_ids)!=len(set(csv_ids)): errors.append('duplicate feature_id in feature-map.csv')
    except Exception as exc: errors.append(f'cannot read feature-map.csv: {exc}')

    try:
        entities=(load_yaml(ROOT/'contracts/database-entity-catalog.yaml') or {}).get('entities') or []
        entity_names=[str(e.get('entity','')) for e in entities]; entity_set=set(entity_names)
        if len(entity_names)!=len(entity_set): errors.append('duplicate entity in database-entity-catalog.yaml')
        for e in entities:
            for fid in e.get('feature_ids') or []:
                if fid not in id_set: errors.append(f'entity {e.get("entity")}: unknown feature ID {fid}')
        for f in feats:
            for entity in [x for x in split_pipe(f.get('data_entities')) if x.lower() not in {'none','n/a'}]:
                if entity not in entity_set: errors.append(f'{f["feature_id"]}: data entity {entity!r} missing from catalog')
    except Exception as exc: errors.append(f'database entity catalog validation failed: {exc}')

    try:
        menus=set((load_yaml(ROOT/'contracts/admin-menu-map.yaml') or {}).get('menus') or [])
        for f in feats:
            menu=str(f.get('admin_control','')).strip()
            if menu.lower() not in {'none','n/a','—','-'} and menu not in menus: errors.append(f'{f["feature_id"]}: admin menu {menu!r} missing from catalog')
    except Exception as exc: errors.append(f'admin menu validation failed: {exc}')

    declared_tests={}
    for f in feats:
        for tid in split_pipe(f.get('test_ids')):
            if tid in declared_tests: errors.append(f'duplicate declared test ID {tid}')
            declared_tests[tid]=f['feature_id']
    try:
        with (ROOT/'contracts/test-catalog.csv').open(encoding='utf-8-sig',newline='') as fh: test_rows=list(csv.DictReader(fh))
        test_ids=[r.get('test_id','') for r in test_rows]
        if len(test_ids)!=len(set(test_ids)): errors.append('duplicate test_id in test-catalog.csv')
        if set(test_ids)!=set(declared_tests): errors.append(f'test catalog differs; missing={sorted(set(declared_tests)-set(test_ids))[:10]}, extra={sorted(set(test_ids)-set(declared_tests))[:10]}')
        for row in test_rows:
            if row.get('feature_id') not in id_set: errors.append(f'test {row.get("test_id")}: unknown feature ID')
            if declared_tests.get(row.get('test_id')) != row.get('feature_id'): errors.append(f'test {row.get("test_id")}: feature mismatch')
    except Exception as exc: errors.append(f'test catalog validation failed: {exc}')

    try:
        with (ROOT/'contracts/api-inventory.csv').open(encoding='utf-8-sig',newline='') as fh: api_rows=list(csv.DictReader(fh))
        for row in api_rows:
            if row.get('feature_id') not in id_set: errors.append(f'API inventory unknown feature ID {row.get("feature_id")}')
            if not str(row.get('endpoint_or_event','')).startswith('/'): errors.append(f'API row {row.get("feature_id")}: endpoint is not HTTP path')
        spec=load_yaml(ROOT/'contracts/openapi-skeleton.yaml') or {}; operations=[]
        for path,path_item in (spec.get('paths') or {}).items():
            for method,operation in (path_item or {}).items():
                if method.lower() in {x.lower() for x in HTTP_METHODS}: operations.append((method.upper(),path,operation or {}))
        op_ids=[o[2].get('operationId') for o in operations]
        if len(op_ids)!=len(set(op_ids)): errors.append('duplicate OpenAPI operationId')
        api_keys=set()
        for r in api_rows:
            for method in split_pipe(r.get('http_method')): api_keys.add((method.upper(),r['endpoint_or_event'],r['feature_id']))
        openapi_keys=set()
        for method,path,op in operations:
            fids=op.get('x-feature-ids') or [op.get('x-feature-id')]
            if not fids: errors.append(f'OpenAPI {op.get("operationId")}: missing x-feature-ids')
            for fid in fids:
                openapi_keys.add((method,path,str(fid)))
                if fid not in id_set: errors.append(f'OpenAPI {op.get("operationId")}: unknown feature ID {fid}')
        if api_keys!=openapi_keys: errors.append(f'OpenAPI/API inventory differ; inventory-only={len(api_keys-openapi_keys)}, openapi-only={len(openapi_keys-api_keys)}')
    except Exception as exc: errors.append(f'API/OpenAPI validation failed: {exc}')

    for phase in PHASES:
        docs=list((ROOT/'phases').glob(f'{phase}_*.md'))
        if len(docs)!=1: errors.append(f'expected one phase document for {phase}, found {len(docs)}'); continue
        text=docs[0].read_text(encoding='utf-8')
        for fid in [f['feature_id'] for f in feats if f['phase']==phase]:
            if fid not in text: errors.append(f'{docs[0].name}: missing Feature ID {fid}')
        status=ROOT/'status'/f'{phase}_FEATURE_STATUS.yaml'
        if not status.exists(): errors.append(f'missing {status.relative_to(ROOT)}')
        else:
            s=load_yaml(status) or {}; actual={e.get('feature_id') for e in (s.get('features') or [])}; expected={f['feature_id'] for f in feats if f['phase']==phase}
            if actual!=expected: errors.append(f'{phase} status IDs differ; missing={sorted(expected-actual)}, extra={sorted(actual-expected)}')

    try:
        root_current=load_yaml(ROOT/'CURRENT_PHASE.yaml')
        if not root_current.get('phase_id'): errors.append('CURRENT_PHASE.yaml missing phase_id')
        root_release=load_yaml(ROOT/'RELEASE_CONTRACT.yaml'); contract_release=load_yaml(ROOT/'contracts/release-contract.yaml')
        if root_release!=contract_release: errors.append('root RELEASE_CONTRACT.yaml differs from contracts copy')
        required=set(root_release.get('required_outputs') or []); contract_required=set(contract_release.get('required_outputs') or [])
        if required!=contract_required: errors.append('root and contracts release output names differ')
        if not {
            '原功能清单.md','功能完成对比清单.md','完整测试清单.md','部署证据.md',
            '截图索引.csv','校验文件_SHA256.txt','服务器构建来源证明.json',
            '本机下载校验证明.json','真机验收证据.json','真机日志审查.md',
            '真实页面截图索引.csv','真实页面视觉审查.json',
        }.issubset(required):
            errors.append('release contract does not enforce all required Chinese owner artifact names')
        forbidden = root_release.get('forbidden') or {}
        if any(forbidden.get(key) is not True for key in ('github_actions','android_emulator','every_other_simulator')):
            errors.append('release contract must forbid GitHub Actions, Android Emulator and every simulator')

        android_acceptance = load_yaml(ROOT/'contracts/android-phase-acceptance.yaml') or {}
        forbidden_execution = android_acceptance.get('forbidden_execution') or {}
        if any(forbidden_execution.get(key) is not True for key in ('github_actions','android_emulator','every_other_simulator')):
            errors.append('Android acceptance contract does not enforce the forbidden execution modes')
        server_build = android_acceptance.get('online_server_build') or {}
        physical = android_acceptance.get('physical_device') or {}
        if server_build.get('exact_clean_git_commit_required') is not True or server_build.get('download_sha256_must_match_server') is not True:
            errors.append('Android acceptance contract must require exact server build and SHA-256 download verification')
        if physical.get('adb_state_required') != 'device' or physical.get('busy_action') != 'choose_shortest_fifo_and_wait_without_interrupting_owner':
            errors.append('Android acceptance contract must require exact device state and shortest shared FIFO waiting')
        if physical.get('automatic_selection') != 'prefer_idle_then_shortest_fifo' or physical.get('explicit_serial_override_allowed') is not True:
            errors.append('Android acceptance contract must define idle-device preference, shortest FIFO selection and explicit serial override')

        version_matrix = load_yaml(ROOT/'contracts/release-version-matrix.yaml') or {}
        global_requirements = version_matrix.get('global_release_requirements') or {}
        if global_requirements.get('applies_to_every_phase') is not True:
            errors.append('release-version-matrix must apply hard gates to every phase')
        if global_requirements.get('build_pipeline') != 'online_server_exact_commit':
            errors.append('release-version-matrix must use the online-server exact-commit build pipeline')
        if global_requirements.get('acceptance_device') != 'owner_physical_android_phone':
            errors.append('release-version-matrix must use the owner physical Android phone')
        if global_requirements.get('github_actions_forbidden') is not True or global_requirements.get('emulators_and_simulators_forbidden') is not True:
            errors.append('release-version-matrix must forbid GitHub Actions and simulators')
        if global_requirements.get('logo_resource') != 'android/app/src/main/res/drawable/ylven_logo.png':
            errors.append('release-version-matrix logo_resource is not the fixed YLVEN logo')
        if global_requirements.get('splash_resource') != 'android/app/src/main/res/drawable-nodpi/ylven_splash.png':
            errors.append('release-version-matrix splash_resource is not the fixed YLVEN startup image')
        upgrade = global_requirements.get('overlay_install') or {}
        if upgrade.get('command') != 'adb install -r' or upgrade.get('clear_data_forbidden') is not True:
            errors.append('release-version-matrix must require adb install -r without clearing data')
        migrations = version_matrix.get('approved_signing_migrations') or {}
        p03_migration = migrations.get('P03') or {}
        migration_path = ROOT / 'contracts' / 'signing-migrations' / 'P03.properties'
        if p03_migration.get('approval_file') != 'contracts/signing-migrations/P03.properties':
            errors.append('release-version-matrix must point P03 to the canonical signing migration approval')
        if p03_migration.get('install_mode') != 'one_time_uninstall_then_install' or p03_migration.get('data_preserved') is not False:
            errors.append('P03 signing migration must explicitly record destructive one-time installation')
        if not migration_path.is_file():
            errors.append('P03 signing migration approval file is missing')
        else:
            props = {}
            for line in migration_path.read_text(encoding='utf-8').splitlines():
                if line and not line.startswith('#') and '=' in line:
                    key, value = line.split('=', 1)
                    props[key] = value
            expected_props = {
                'phase': 'P03', 'approved': 'true', 'approved_by': 'project_owner',
                'install_mode': 'one_time_uninstall_then_install', 'data_preserved': 'false',
                'future_upgrade_baseline': 'P03',
            }
            for key, value in expected_props.items():
                if props.get(key) != value:
                    errors.append(f'P03 signing migration property {key} mismatch')
            for key in ('previous_certificate_sha256', 'new_certificate_sha256'):
                if not re.fullmatch(r'[0-9a-f]{64}', props.get(key, '')):
                    errors.append(f'P03 signing migration property {key} is not a SHA-256 digest')
        chinese = global_requirements.get('chinese_delivery_files') or {}
        required_chinese = {
            'original_features': '原功能清单.md',
            'completion_comparison': '功能完成对比清单.md',
            'complete_test_checklist': '完整测试清单.md',
            'deployment_evidence': '部署证据.md',
            'upgrade_evidence': '覆盖安装证据.md',
            'screenshots': '截图',
            'screenshot_index': '截图索引.csv',
            'production_screenshot_index': '真实页面截图索引.csv',
            'production_visual_review': '真实页面视觉审查.json',
            'checksum': '校验文件_SHA256.txt',
        }
        for key, expected in required_chinese.items():
            if chinese.get(key) != expected:
                errors.append(f'release-version-matrix Chinese delivery file {key} must be {expected}')
        admin = global_requirements.get('admin_real_test') or {}
        if admin.get('codex_must_open_every_version') is not True or admin.get('current_phase_features_must_be_real_and_complete') is not True:
            errors.append('release-version-matrix must require Codex real admin testing for every version')
        phase_rows = version_matrix.get('phases') or []
        by_phase = {str(row.get('phase')): row for row in phase_rows}
        if set(by_phase) != set(PHASES):
            errors.append('release-version-matrix phase rows must cover exactly P00-P13')
        for index, phase_id in enumerate(PHASES):
            row = by_phase.get(phase_id) or {}
            if row.get('admin_access_delivery') is not True:
                errors.append(f'{phase_id}: admin_access_delivery must be true; every version requires real admin testing')
            if row.get('release_pipeline') != 'online-server-build-and-physical-device-acceptance':
                errors.append(f'{phase_id}: release_pipeline must use online server and physical device acceptance')
            expected_upgrade = None if index == 0 else str((by_phase.get(PHASES[index - 1]) or {}).get('version_name'))
            if row.get('upgrade_test_from') != expected_upgrade:
                errors.append(f'{phase_id}: upgrade_test_from must be {expected_upgrade!r}')
    except Exception as exc: errors.append(f'root state/release contract validation failed: {exc}')

    if len((ROOT/'AGENTS.md').read_bytes())>32768: errors.append('AGENTS.md exceeds 32 KiB')
    phase=args.phase or (load_yaml(ROOT/'CURRENT_PHASE.yaml') or {}).get('current_phase')
    if phase and phase not in PHASES: errors.append(f'unknown phase {phase}')
    if args.release and phase:
        s=load_yaml(ROOT/'status'/f'{phase}_FEATURE_STATUS.yaml') or {}; entries=s.get('features') or []
        for e in entries:
            st=e.get('status')
            if st not in FINAL_STATUSES: errors.append(f'{e.get("feature_id")}: release status {st!r} is not final')
            if st!='IMPLEMENTED' and not str(e.get('reason','')).strip(): errors.append(f'{e.get("feature_id")}: non-implemented status requires reason')
            if st=='IMPLEMENTED' and not (e.get('evidence') or []): errors.append(f'{e.get("feature_id")}: IMPLEMENTED requires evidence')

    if errors:
        for error in errors: print('ERROR:',error,file=sys.stderr)
        return 1
    selected=sum(1 for f in feats if not phase or f['phase']==phase)
    print(f'Contract validation passed: {len(feats)} total features; {selected} selected for {phase or "all"}; {len(declared_tests)} tests declared.')
    return 0
if __name__=='__main__': raise SystemExit(main())
