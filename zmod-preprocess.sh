#!/bin/bash
# Lite-версия постпроцессора: только zmod_color_data + MD5, без упрощения
# EXCLUDE_OBJECT_DEFINE (для этого используйте Python- или Go-версию).
set -u

FILE_PATH="${1:-}"

if [[ -z "$FILE_PATH" || ! -f "$FILE_PATH" ]]; then
    echo "Usage: $0 <gcode_file>" >&2
    exit 1
fi

TEMP_CONTENT=$(mktemp)
FINAL_TEMP=$(mktemp)
trap 'rm -f "$TEMP_CONTENT" "$FINAL_TEMP"' EXIT

# 1. Убираем старые MD5/zmod-строки и нормализуем переводы строк (как Python/Go)
grep -aviE '^; (MD5:|zmod_color_data =)' "$FILE_PATH" | tr -d '\r' > "$TEMP_CONTENT" || true

# 2. Номера инструментов в порядке первого появления (как в Python/Go)
TOOL_IDS=$(awk '{
    l = tolower($0); sub(/^[ \t]+/, "", l)
    split(l, a, /[ ;]/); tok = a[1]
    if (tok ~ /^t[0-9]+$/) { n = substr(tok, 2) + 0; if (!seen[n]++) out = out (out == "" ? "" : ",") n }
} END { print out }' "$TEMP_CONTENT")
[[ -z "$TOOL_IDS" ]] && TOOL_IDS="0"

# 3. Цвета и типы филамента: берём последнюю строку, нижний регистр, ';' -> ','
FILAMENT_COLOURS=$( { grep -ai "^; filament_colour =" "$TEMP_CONTENT" || true; } | tail -n 1 | cut -d'=' -f2- | tr -d ' ' | tr ';' ',' | tr '[:upper:]' '[:lower:]')
FILAMENT_TYPES=$( { grep -ai "^; filament_type =" "$TEMP_CONTENT" || true; } | tail -n 1 | cut -d'=' -f2- | tr -d ' ' | tr ';' ',' | tr '[:upper:]' '[:lower:]')

ZMOD_LINE="; zmod_color_data = ${TOOL_IDS}|${FILAMENT_COLOURS}|${FILAMENT_TYPES}"

# 4. Вставляем перед первым HEADER_BLOCK_END; если его нет — в начало файла.
#    awk вместо sed 'i': sed 'i text' не работает в BSD/macOS sed.
if grep -qai "^; header_block_end" "$TEMP_CONTENT"; then
    awk -v line="$ZMOD_LINE" '!ins && tolower($0) ~ /^; header_block_end/ { print line; ins = 1 } { print }' "$TEMP_CONTENT" > "$FINAL_TEMP"
else
    { printf '%s\n' "$ZMOD_LINE"; cat "$TEMP_CONTENT"; } > "$FINAL_TEMP"
fi

# 5. MD5 от финальных байтов (разделитель \n, как в Python/Go — MD5 считается
#    по фактическому содержимому, поэтому CRLF не нужен)
if command -v md5sum >/dev/null; then
    MD5_HASH=$(md5sum "$FINAL_TEMP" | awk '{print $1}')
else
    MD5_HASH=$(md5 -q "$FINAL_TEMP")
fi

# 6. Дописываем MD5-заголовок
{ printf '; MD5:%s\n' "$MD5_HASH"; cat "$FINAL_TEMP"; } > "$FILE_PATH"
