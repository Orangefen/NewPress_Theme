"""Extract decorative assets from the user-supplied DocsHub design reference.

Usage: python scripts/prepare_docshub_assets.py --reference path/to/reference.png
The reference is an image asset, never executable input. UI is implemented in templates.
"""
import argparse
from pathlib import Path
import cv2
import numpy as np
from PIL import Image, ImageFilter

parser = argparse.ArgumentParser()
parser.add_argument('--reference', required=True, type=Path)
args = parser.parse_args()
source = Image.open(args.reference).convert('RGB')
if source.size != (1254, 1254):
    parser.error('expected the original 1254 × 1254 reference')
target = Path(__file__).resolve().parents[1] / 'sources/docshub/assets/images'
target.mkdir(parents=True, exist_ok=True)
hero = np.array(source.crop((218, 0, 1254, 481)))
mask = np.zeros(hero.shape[:2], np.uint8)
# Remove the reference's text, search input and top navigation. Keep the glass artwork.
for x1, y1, x2, y2 in [(27, 14, 490, 72), (766, 14, 1010, 74), (34, 128, 443, 411)]:
    mask[y1:y2, x1:x2] = 255
clean = cv2.inpaint(hero, mask, 18, cv2.INPAINT_TELEA)
image = Image.fromarray(clean)
# Broad background areas use soft light without inpainting edge streaks.
blur = image.filter(ImageFilter.GaussianBlur(24))
soft_mask = Image.fromarray(mask).filter(ImageFilter.GaussianBlur(12))
image = Image.composite(blur, image, soft_mask)
image.save(target / 'glass-ribbon.webp', quality=94, method=6)
for name, box in {
    'tutorial-workspace': (253, 658, 481, 742),
    'tutorial-components': (497, 658, 724, 742),
    'tutorial-code': (742, 658, 969, 742),
    'tutorial-performance': (986, 658, 1213, 742),
}.items():
    source.crop(box).save(target / (name + '.webp'), quality=95, method=6)
community = np.array(source.crop((934, 902, 1217, 1042)))
mask = np.zeros(community.shape[:2], np.uint8)
for x1, y1, x2, y2 in [(12, 25, 169, 84), (9, 87, 119, 127), (238, 14, 272, 44)]:
    mask[y1:y2, x1:x2] = 255
Image.fromarray(cv2.inpaint(community, mask, 8, cv2.INPAINT_TELEA)).save(target / 'community-mountains.webp', quality=94, method=6)
print('DocsHub: 6 local decorative assets extracted')
