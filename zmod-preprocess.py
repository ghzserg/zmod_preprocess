#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# (C) Namida Verasche aka ninjamida
# MD5 aspects copied (with changes) from addMD5.py

import sys
import json
import argparse
import math
import hashlib
import os


# Object outline simplification

def format_coord(value):
    # Go writes coordinates with strconv.FormatFloat(v, 'f', -1, 64): the shortest
    # form that round-trips, never exponential. repr() picks the same digits, but
    # keeps a trailing '.0' and falls back to an exponent on very small values.
    text = repr(value)
    if 'e' in text or 'E' in text:
        text = ('%.17f' % value).rstrip('0').rstrip('.')
        if text in ('', '-'):
            text = '-0' if math.copysign(1, value) < 0 else '0'
    elif text.endswith('.0'):
        text = text[:-2]
    return text


def douglas_peucker(pts, tolerance):
    # Keeps the points of an arc that stray from its chord further than the
    # tolerance. Both ends are always kept.
    #
    # Split with an explicit stack rather than recursion: an outline holds
    # thousands of points and nothing bounds the depth of the split, while
    # Python stops at a thousand frames.
    if len(pts) < 3:
        return list(pts)
    keep = [False] * len(pts)
    keep[0] = keep[-1] = True
    stack = [(0, len(pts) - 1)]
    while stack:
        first, last = stack.pop()
        if last - first < 2:
            continue
        ax, ay = pts[first]
        bx, by = pts[last]
        dx, dy = bx - ax, by - ay
        chord = math.hypot(dx, dy)
        worst, at = 0.0, 0
        for i in range(first + 1, last):
            px, py = pts[i]
            if chord == 0:
                dist = math.hypot(px - ax, py - ay)
            else:
                dist = abs(dx * (ay - py) - dy * (ax - px)) / chord
            if dist > worst:
                worst, at = dist, i
        if worst > tolerance:
            keep[at] = True
            stack.append((first, at))
            stack.append((at, last))
    return [pts[i] for i in range(len(pts)) if keep[i]]


def simplify_ring(pts, tolerance):
    # Thins out a closed outline. The points extreme along each axis are pinned,
    # so the bounding box never moves. A zero tolerance leaves that box alone.
    min_x = max_x = min_y = max_y = 0
    for i, (x, y) in enumerate(pts):
        if x < pts[min_x][0]:
            min_x = i
        if x > pts[max_x][0]:
            max_x = i
        if y < pts[min_y][1]:
            min_y = i
        if y > pts[max_y][1]:
            max_y = i
    if tolerance <= 0:
        x0, y0 = pts[min_x][0], pts[min_y][1]
        x1, y1 = pts[max_x][0], pts[max_y][1]
        return [(x0, y0), (x1, y0), (x1, y1), (x0, y1)]

    anchors = sorted(set([min_x, max_x, min_y, max_y]))
    if len(anchors) < 2:
        return pts

    out = []
    for i, a in enumerate(anchors):
        b = anchors[(i + 1) % len(anchors)]
        if b > a:
            arc = pts[a:b + 1]
        else:
            arc = pts[a:] + pts[:b + 1]  # the last arc runs through the start
        kept = douglas_peucker(arc, tolerance)
        out += kept[:-1]  # the end of an arc is the start of the next one
    return out


POLYGON_DECODER = json.JSONDecoder()


def simplify_polygon(line, tolerance):
    # Thins out the outline in EXCLUDE_OBJECT_DEFINE. The slicer writes the whole
    # first-layer outline of every object, and on a plate of many small parts that
    # runs to thousands of points - parsing them stalls Klipper on the printer.
    # Cancelling an object goes by name and centre, and KAMP and LINE_PURGE take
    # only the bounding box from the polygon, so the box is kept exact at any
    # tolerance. Returns None when there is nothing to replace.
    start = line.find('POLYGON=')
    if start < 0:
        return None
    start += len('POLYGON=')

    # The value is plain JSON, and raw_decode also reports where it ended - the
    # line carries the rest of the command after it. Anything that does not parse
    # is left alone: an untouched outline beats one rebuilt from points we failed
    # to read.
    try:
        raw, end = POLYGON_DECODER.raw_decode(line, start)
    except ValueError:
        return None
    if not isinstance(raw, list) or len(raw) < 5:
        return None
    pts = []
    for pair in raw:
        if not isinstance(pair, list) or len(pair) != 2:
            return None
        for value in pair:
            if isinstance(value, bool) or not isinstance(value, (int, float)):
                return None
        pts.append((float(pair[0]), float(pair[1])))

    simple = simplify_ring(pts, tolerance)
    if len(simple) >= len(pts):
        return None

    body = '[' + ','.join('[%s,%s]' % (format_coord(x), format_coord(y))
                          for x, y in simple) + ']'
    return line[:start] + body + line[end:]


# Arguments: the file path stays positional, the keys are optional. The slicer
# appends the path last, so the keys always come before it.

parser = argparse.ArgumentParser(add_help=False)
parser.add_argument('file_path', nargs='?')
parser.add_argument('-simplify-objects', '--simplify-objects', type=float,
                    default=0.2, dest='tolerance',
                    help='outline tolerance in mm; 0 leaves the bounding rectangle')
parser.add_argument('-no-simplify-objects', '--no-simplify-objects',
                    action='store_true', dest='no_simplify',
                    help='keep the full outline')
args = parser.parse_args()

if args.tolerance < 0:
    sys.stderr.write('Bad tolerance: %s\n' % args.tolerance)
    sys.exit(1)

file_path = args.file_path
simplify = not args.no_simplify
tolerance = args.tolerance

if not file_path:
    sys.exit()

# Color info

with open(file_path, 'r', encoding='utf-8') as f:
    content = f.readlines()

if content[0].strip().casefold().startswith('; md5'):
    content.pop(0)

result_colors = []
highest_result_color = -1
filament_color_line = ''
filament_type_line = ''

color_data_line = ''

header_end_line = -1
remove_existing_line = -1
found_header_end_line = False
found_existing_line = False
old_color_data_line = -1

for line_index, line_raw in enumerate(content):
    if simplify and 'EXCLUDE_OBJECT_DEFINE' in line_raw:
        simplified = simplify_polygon(line_raw, tolerance)
        if simplified is not None:
            content[line_index] = simplified
            line_raw = simplified

    line = line_raw.strip().casefold()
    if not found_header_end_line:
        header_end_line += 1
    if not found_existing_line:
        remove_existing_line += 1
    if len(line) == 0:
        continue
    if line[0] == 't':
        main_part = line.split(';', 1)[0].split(' ', 1)[0].strip()
        try:
            index = int(main_part[1:])
            if index not in result_colors:
                result_colors += [index]
            highest_result_color = max(highest_result_color, index)
        except:
            pass
    if line[0] == ';':
        if line.startswith('; filament_colour ='):
            _, _, filament_color_line = line.partition('=')
        if line.startswith('; filament_type ='):
            _, _, filament_type_line = line.partition('=')
        if line.startswith('; zmod_color_data ='):
            found_existing_line = True
        if line.startswith('; header_block_end'):
            found_header_end_line = True

filament_colors = filament_color_line.strip().split(';')
filament_types = filament_type_line.strip().split(';')

if filament_colors[0] == '':
    filament_colors = []
if filament_types[0] == '':
    filament_types = []

if len(result_colors) == 0:
    result_colors = [0]
    highest_result_color = 0

if len(filament_colors) <= highest_result_color:
    filament_colors += [''] * (highest_result_color + 1 - len(filament_colors))

if len(filament_types) <= highest_result_color:
    filament_types += [''] * (highest_result_color + 1 - len(filament_types))

tool_indexes_string = ','.join([str(result_color) for result_color in result_colors])
filament_color_string = ','.join(filament_colors)
filament_type_string = ','.join(filament_types)

if not found_header_end_line:
    header_end_line = 0

content.insert(header_end_line, f"; zmod_color_data = {tool_indexes_string}|{filament_color_string}|{filament_type_string}\r\n")

if found_existing_line:
    if remove_existing_line > header_end_line: # Should never happen but just in case
        remove_existing_line += 1
    content.pop(remove_existing_line)

# MD5

content = "".join(content).encode('utf-8')

md5_hash = hashlib.md5(content).hexdigest()

md5_line = b'; MD5:' + md5_hash.encode('ascii') + b'\r\n'

new_content = md5_line + content

with open(file_path, 'wb') as f:
    f.write(new_content)
