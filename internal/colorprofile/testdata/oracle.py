#!/usr/bin/env python3
"""Regenerate original ICC fixtures and numeric reference vectors with LittleCMS.

This optional maintainer script transforms RGB channel arrays. It does not read
or edit images. The Go tests use the saved fixtures and need no native library.
Set LCMS_LIBRARY when the platform cannot find the installed shared library.
"""

import ctypes as C
import ctypes.util
import json
import os
from pathlib import Path
import struct


class xyY(C.Structure):
    _fields_ = [("x", C.c_double), ("y", C.c_double), ("Y", C.c_double)]


class Primaries(C.Structure):
    _fields_ = [("red", xyY), ("green", xyY), ("blue", xyY)]


library = os.environ.get("LCMS_LIBRARY") or ctypes.util.find_library("lcms2")
if not library:
    raise SystemExit("Install LittleCMS or set LCMS_LIBRARY to its shared library.")
lcms = C.CDLL(library)


def function(name, result, args):
    fn = getattr(lcms, name)
    fn.restype = result
    fn.argtypes = args
    return fn


pointer = C.c_void_p
build_gamma = function("cmsBuildGamma", pointer, [pointer, C.c_double])
build_curve = function("cmsBuildParametricToneCurve", pointer, [pointer, C.c_int, C.POINTER(C.c_double)])
free_curve = function("cmsFreeToneCurve", None, [pointer])
create_rgb = function("cmsCreateRGBProfile", pointer, [C.POINTER(xyY), C.POINTER(Primaries), C.POINTER(pointer)])
create_srgb = function("cmsCreate_sRGBProfile", pointer, [])
set_version = function("cmsSetProfileVersion", None, [pointer, C.c_double])
save = function("cmsSaveProfileToMem", C.c_int, [pointer, pointer, C.POINTER(C.c_uint32)])
open_profile = function("cmsOpenProfileFromMem", pointer, [pointer, C.c_uint32])
close = function("cmsCloseProfile", C.c_int, [pointer])
create_transform = function("cmsCreateTransform", pointer, [pointer, C.c_uint32, pointer, C.c_uint32, C.c_uint32, C.c_uint32])
transform_rgb = function("cmsDoTransform", None, [pointer, pointer, pointer, C.c_uint32])
delete_transform = function("cmsDeleteTransform", None, [pointer])
encoded_version = function("cmsGetEncodedCMMversion", C.c_int, [])

directory = Path(__file__).resolve().parent
white = xyY(.3127, .3290, 1)
rgb16 = (4 << 16) | (3 << 3) | 2
samples = [
    [0, 0, 0], [65535, 65535, 65535], [32768, 32768, 32768],
    [65535, 0, 0], [0, 65535, 0], [0, 0, 65535],
    [1000, 2000, 3000], [16384, 32768, 49152],
    [49152, 16384, 32768], [65535, 32768, 8192],
    [28385, 32839, 24870], [28895, 32748, 24516],
    [60000, 13126, 9081], [500, 500, 500], [12345, 23456, 34567],
]
profiles = [
    ("srgb", (.64, .33, .30, .60, .15, .06), 4.4, None),
    ("display-p3", (.68, .32, .265, .69, .15, .06), 4.4, None),
    ("adobe-rgb", (.64, .33, .21, .71, .15, .06), 2.1, 563 / 256),
]
oracle = {"littlecms_version": encoded_version(), "intent": "relative-colorimetric", "flags": "NOOPTIMIZE", "profiles": []}
destination = create_srgb()
if not destination:
    raise SystemExit("LittleCMS could not create its sRGB destination.")
for name, coordinates, version, gamma in profiles:
    if gamma is None:
        # LittleCMS type 4 corresponds to ICC parametricCurveType 3.
        params = (C.c_double * 5)(2.4, 1 / 1.055, .055 / 1.055, 1 / 12.92, .04045)
        curve = build_curve(None, 4, params)
    else:
        curve = build_gamma(None, gamma)
    primary = Primaries(*(xyY(coordinates[i], coordinates[i + 1], 1) for i in range(0, 6, 2)))
    profile = create_rgb(C.byref(white), C.byref(primary), (pointer * 3)(curve, curve, curve))
    free_curve(curve)
    if not profile:
        raise SystemExit(f"LittleCMS could not create {name}.")
    set_version(profile, version)
    size = C.c_uint32()
    if not save(profile, None, C.byref(size)):
        raise SystemExit(f"LittleCMS could not size {name}.")
    raw = C.create_string_buffer(size.value)
    if not save(profile, raw, C.byref(size)):
        raise SystemExit(f"LittleCMS could not serialize {name}.")
    close(profile)
    data = bytearray(raw.raw)
    # Fix the creation time and clear the optional profile ID for reproducibility.
    data[24:36] = struct.pack(">6H", 2026, 10, 3, 0, 0, 0)
    data[84:100] = bytes(16)
    filename = name + ".icc"
    (directory / filename).write_bytes(data)
    # Reopen the serialized bytes so quantized ICC parameters drive the oracle.
    source_buffer = C.create_string_buffer(bytes(data))
    source = open_profile(source_buffer, len(data))
    conversion = create_transform(source, rgb16, destination, rgb16, 1, 0x0100)
    if not conversion:
        raise SystemExit(f"LittleCMS could not transform {name}.")
    reference = []
    for values in samples:
        source_rgb = (C.c_uint16 * 3)(*values)
        destination_rgb = (C.c_uint16 * 3)()
        transform_rgb(conversion, source_rgb, destination_rgb, 1)
        reference.append({"input": values, "srgb": list(destination_rgb)})
    delete_transform(conversion)
    close(source)
    oracle["profiles"].append({"id": name, "file": filename, "vectors": reference})
close(destination)
(directory / "littlecms-vectors.json").write_text(json.dumps(oracle, indent=2) + "\n")
print(f"Generated {len(profiles)} original profiles and {len(samples) * len(profiles)} LittleCMS vectors.")
