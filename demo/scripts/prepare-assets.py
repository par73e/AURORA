#!/usr/bin/env python3
"""Download public image assets on the Mac; server runtime never contacts CDNs.
Requires curl + macOS sips. Author/source/license metadata stays in the JSON.
Failed images are omitted; a completely failed run preserves the prior wall.
"""
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import hashlib, json, re, subprocess, os
ROOT = Path(__file__).resolve().parents[1]
PUBLIC = ROOT / 'frontend/public'
IMAGES = PUBLIC / 'demo-images'
IMAGES.mkdir(parents=True, exist_ok=True)

def download(url, target):
    if target.exists():
        return
    temporary = target.with_suffix(target.suffix + '.part')
    try:
        subprocess.run(['curl', '-fLsS', '--retry', '1', '--max-time', '40', '--max-filesize', '30000000', url, '-o', str(temporary)], check=True, capture_output=True)
        # Decode through the system image tool, rather than accepting HTTP 200 HTML.
        details = subprocess.check_output(['sips', '-g', 'pixelWidth', '-g', 'pixelHeight', str(temporary)], text=True, stderr=subprocess.STDOUT)
        dimensions = [int(x) for x in re.findall(r'pixel(?:Width|Height): (\d+)', details)]
        if len(dimensions) != 2 or min(dimensions) < 100:
            raise ValueError('not a useful decoded image')
        temporary.replace(target)
    finally:
        temporary.unlink(missing_ok=True)

def prepare(item):
    if item.get('status') != 'ready' or item.get('mediaType') != 'image': return None
    url = item.get('imageUrl', '')
    if not url.startswith('https://') or 'nasa-logo' in url.lower() or item.get('title') == 'NASA Science': return None
    filename = hashlib.sha256(url.encode()).hexdigest()[:18] + '.jpg'
    target = IMAGES / filename
    try:
        if not target.exists():
            original = IMAGES / (filename + '.original')
            download(url, original)
            subprocess.run(['sips', '-s', 'format', 'jpeg', '-s', 'formatOptions', '83', '-Z', '1800', str(original), '--out', str(target)], check=True, capture_output=True)
            original.unlink(missing_ok=True)
        local = '/aurora/demo-images/' + filename
        result = dict(item, imageUrl=local, thumbnailUrl=local, hdUrl=local, selectionMode='curated', isFallback=False)
        print('Saved:', item['title'], flush=True)
        return result
    except Exception as error:
        print('Omitted:', item.get('title'), type(error).__name__, flush=True)
        target.unlink(missing_ok=True)
        return None

wall = json.loads((ROOT / 'data/image-wall-remote.json').read_text())
items = wall['recent'] + wall['collection']
with ThreadPoolExecutor(max_workers=4) as pool:
    saved = list(pool.map(prepare, items))
recent = [x for x in saved[:len(wall['recent'])] if x]
collection = [x for x in saved[len(wall['recent']):] if x]
if not recent and not collection:
    raise SystemExit('No usable images; previous local image wall left untouched.')
local_wall = dict(wall, recent=recent, collection=collection)
output = ROOT / 'data/image-wall.json'
temporary = output.with_suffix('.json.tmp')
temporary.write_text(json.dumps(local_wall, ensure_ascii=False, indent=2))
os.replace(temporary, output)
download('https://unpkg.com/three-globe@2.45.2/example/img/earth-night.jpg', PUBLIC / 'earth-night.jpg')
print('Local image wall:', len(recent), 'archive images +', len(collection), 'collection images')
