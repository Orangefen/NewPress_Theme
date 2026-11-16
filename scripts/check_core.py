"""Run theme contracts and integration tests without copying files into core."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile

parser = argparse.ArgumentParser()
parser.add_argument('--core', required=True, type=Path)
parser.add_argument('--ui', action='store_true', help='start the opt-in Nexora browser fixture')
args = parser.parse_args()
root = Path(__file__).resolve().parents[1]
core = args.core.resolve()
if not (core / 'go.mod').is_file():
    parser.error('--core must point to a NewPress checkout')
environment = os.environ.copy()
environment['NEWPRESS_NEXORA_SOURCE'] = str(root / 'sources/nexora')
if args.ui:
    environment['NEWPRESS_NEXORA_UI'] = '1'
else:
    subprocess.run(['go', 'run', './cmd/themecheck', '--source', str(root / 'sources/nexora')], cwd=core, check=True, env=environment)
with tempfile.TemporaryDirectory(prefix='newpress-theme-tests-') as temporary:
    overlay = Path(temporary) / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(core / 'internal/httpserver/nexora_external_test.go'): str(root / 'tests/integration/nexora_theme_test.go')}}), encoding='utf-8')
    pattern = '^TestNexoraInteractiveUI$' if args.ui else '^TestNexoraPublicTheme$'
    subprocess.run(['go', 'test', '-overlay', str(overlay), './internal/httpserver', '-run', pattern, '-count=1', '-v'], cwd=core, check=True, env=environment)
