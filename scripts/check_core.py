"""Run theme contracts and integration tests without copying files into core."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile

parser = argparse.ArgumentParser()
parser.add_argument('--core', required=True, type=Path)
parser.add_argument('--ui', action='store_true', help='start the selected theme in an isolated browser fixture')
parser.add_argument('--theme', choices=['nexora', 'docshub'], default='nexora')
args = parser.parse_args()
root = Path(__file__).resolve().parents[1]
core = args.core.resolve()
if not (core / 'go.mod').is_file():
    parser.error('--core must point to a NewPress checkout')
environment = os.environ.copy()
slug = args.theme
label = {'nexora': 'Nexora', 'docshub': 'DocsHub'}[slug]
environment['NEWPRESS_' + slug.upper() + '_SOURCE'] = str(root / 'sources' / slug)
if args.ui:
    environment['NEWPRESS_' + slug.upper() + '_UI'] = '1'
else:
    subprocess.run(['go', 'run', './cmd/themecheck', '--source', str(root / 'sources' / slug)], cwd=core, check=True, env=environment)
with tempfile.TemporaryDirectory(prefix='newpress-theme-tests-') as temporary:
    overlay = Path(temporary) / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(core / f'internal/httpserver/{slug}_external_test.go'): str(root / f'tests/integration/{slug}_theme_test.go')}}), encoding='utf-8')
    pattern = '^Test' + label + ('InteractiveUI$' if args.ui else 'PublicTheme$')
    subprocess.run(['go', 'test', '-overlay', str(overlay), './internal/httpserver', '-run', pattern, '-count=1', '-timeout=60m' if args.ui else '-timeout=5m', '-v'], cwd=core, check=True, env=environment)
