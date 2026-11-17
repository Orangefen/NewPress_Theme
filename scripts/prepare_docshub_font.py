"""Build the local DocsHub font from the recorded official Noto Sans SC source.

Requires fonttools and brotli. Pass the downloaded NotoSansSC-VF.ttf and its OFL
license; this helper does not download files or modify its input.
"""
import argparse
import hashlib
from pathlib import Path
import shutil
from fontTools.ttLib import TTFont
from fontTools import subset

parser = argparse.ArgumentParser()
parser.add_argument('--source', required=True, type=Path)
parser.add_argument('--license', required=True, type=Path)
args = parser.parse_args()
expected = 'd68bafcb48a2707749396aa12bbbd833cb70401f3a9a689fd2902c7e0d295964'
if hashlib.sha256(args.source.read_bytes()).hexdigest() != expected:
    parser.error('font source differs from the recorded Noto Sans SC 2.004 file')
chars = set(chr(i) for i in range(32, 127))
for lead in range(0xa1, 0xf8):
    for trail in range(0xa1, 0xff):
        try:
            chars.update(bytes([lead, trail]).decode('gb2312'))
        except UnicodeDecodeError:
            pass
for first, last in [(0x2000, 0x2070), (0x3000, 0x3040), (0xff00, 0xfff0)]:
    chars.update(chr(i) for i in range(first, last))
font = TTFont(args.source)
# Some static glyphs have no variation entry in the upstream variable font.
for name in font.getGlyphOrder():
    if name not in font['gvar'].variations:
        font['gvar'].variations[name] = []
options = subset.Options()
options.flavor = 'woff2'
options.recalc_timestamp = False
subsetter = subset.Subsetter(options=options)
subsetter.populate(unicodes=[ord(c) for c in chars])
subsetter.subset(font)
font.flavor = 'woff2'
target = Path(__file__).resolve().parents[1] / 'sources/docshub/assets/fonts'
target.mkdir(parents=True, exist_ok=True)
font.save(target / 'docshub-sans.woff2')
shutil.copyfile(args.license, target / 'OFL.txt')
print('DocsHub local WOFF2 font built')
