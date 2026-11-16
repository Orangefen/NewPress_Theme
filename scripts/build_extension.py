#!/usr/bin/env python3
"""Build a deterministic plugin/theme ZIP from a clean source directory.

The signed catalog publisher performs the authoritative manifest/runtime checks.
This helper never signs or publishes and refuses installed/private state.
"""
import argparse
import json
import os
from pathlib import Path
import stat
import tempfile
import zipfile


RESERVED = {"data", ".official.json", ".integrity.json", ".disabled",
            ".versions.json", ".rollback", ".review-required",
            ".rollback-configuration.json"}
SKIP = {".git", "node_modules", "__pycache__"}


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f"duplicate JSON key: {key}")
        result[key] = value
    return result


def build(source: Path, output: Path, kind: str):
    if source.is_symlink() or not source.is_dir():
        raise ValueError("source must be a real directory")
    source = source.resolve()
    output = output.absolute()
    if output.resolve().is_relative_to(source):
        raise ValueError("output must be outside the source directory")
    if output.exists() or output.is_symlink():
        raise ValueError("output already exists")
    for name in RESERVED:
        if (source / name).exists() or (source / name).is_symlink():
            raise ValueError(f"source contains private/installed state: {name}")
    manifest_path = source / f"{kind}.json"
    if manifest_path.is_symlink() or manifest_path.stat().st_size > 1 << 20:
        raise ValueError("manifest is linked or too large")
    manifest = json.loads(manifest_path.read_text("utf-8"), object_pairs_hook=unique_object)
    if manifest.get("schema") != 1 or not manifest.get("version"):
        raise ValueError("a versioned schema-1 manifest is required")
    files, total = [], 0
    for directory, names, filenames in os.walk(source, followlinks=False):
        current = Path(directory)
        for name in names + filenames:
            path = current / name
            if path.is_symlink():
                raise ValueError(f"symbolic link is not allowed: {path.relative_to(source)}")
        names[:] = sorted(name for name in names if name not in SKIP)
        for filename in sorted(filenames):
            path = current / filename
            info = path.stat()
            if not stat.S_ISREG(info.st_mode) or info.st_size > 32 << 20:
                raise ValueError(f"non-regular or oversized file: {path.relative_to(source)}")
            total += info.st_size
            files.append((path, info.st_size))
            if total > 128 << 20 or len(files) > 2000:
                raise ValueError("package exceeds 128 MiB or 2000 files")
    output.parent.mkdir(parents=True, exist_ok=True)
    descriptor, temporary = tempfile.mkstemp(prefix=".extension-", suffix=".zip", dir=output.parent)
    os.close(descriptor)
    try:
        with zipfile.ZipFile(temporary, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
            for path, expected_size in sorted(files):
                data = path.read_bytes()
                if len(data) != expected_size:
                    raise ValueError("source changed during build")
                info = zipfile.ZipInfo(path.relative_to(source).as_posix(), (1980, 1, 1, 0, 0, 0))
                info.create_system = 3
                info.external_attr = (stat.S_IFREG | 0o600) << 16
                archive.writestr(info, data, compress_type=zipfile.ZIP_DEFLATED, compresslevel=9)
        limit = (32 if kind == "plugin" else 64) << 20
        if os.path.getsize(temporary) > limit:
            raise ValueError("compressed package exceeds download limit")
        # Same-filesystem hard linking publishes without ever replacing a file
        # another build may have created after the initial existence check.
        os.link(temporary, output)
    finally:
        os.unlink(temporary)
    print(f"Built {kind} {manifest.get('slug')} {manifest['version']}: {output}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--kind", required=True, choices=("plugin", "theme"))
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    try:
        build(args.source, args.output, args.kind)
    except (OSError, ValueError, json.JSONDecodeError) as error:
        parser.exit(1, f"Build failed: {error}\n")


if __name__ == "__main__":
    main()
