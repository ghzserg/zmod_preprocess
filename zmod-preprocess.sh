#!/bin/bash
# Lite-версия постпроцессора: только zmod_color_data + MD5, без упрощения
# EXCLUDE_OBJECT_DEFINE (для этого используйте Python- или Go-версию).
set -u

# Ключи принимаются в любом месте аргументов (как в Python/Go). Упрощение
# контуров shell-версия не делает: -no-simplify-objects — no-op,
# -simplify-objects — предупреждение; значение проверяется как в Python/Go.
USAGE="Usage: $0 [-simplify-objects=tolerance_mm] [-no-simplify-objects] <gcode_file>"
FILE_PATH=""
TOL=""
TOL_SET=0
while [[ $# -gt 0 ]]; do
    case "$1" in
        -no-simplify-objects|--no-simplify-objects) ;;
        -simplify-objects|--simplify-objects)
            if [[ $# -lt 2 ]]; then
                echo "Error: -simplify-objects expects a value" >&2
                exit 2
            fi
            shift
            TOL="$1"; TOL_SET=1
            ;;
        -simplify-objects=*|--simplify-objects=*)
            TOL="${1#*=}"; TOL_SET=1
            ;;
        -*)
            echo "Unknown option: $1" >&2
            exit 2
            ;;
        *)
            if [[ -n "$FILE_PATH" ]]; then
                echo "$USAGE" >&2
                exit 2
            fi
            FILE_PATH="$1"
            ;;
    esac
    shift
done

if [[ -z "$FILE_PATH" || ! -f "$FILE_PATH" ]]; then
    echo "$USAGE" >&2
    exit 1
fi

if [[ $TOL_SET -eq 1 ]]; then
    verdict=$(awk -v t="$TOL" 'BEGIN {
        l = tolower(t)
        if (l ~ /^[+-]?(nan|inf|infinity)$/) { print "bad"; exit }
        if (t !~ /^[+-]?([0-9]+\.?[0-9]*|\.[0-9]+)([eE][+-]?[0-9]+)?$/) { print "invalid"; exit }
        if (t + 0 < 0) { print "bad"; exit }
        print "ok"
    }')
    case "$verdict" in
        bad)
            echo "Bad tolerance: $TOL" >&2
            exit 1
            ;;
        invalid)
            echo "Error: invalid -simplify-objects value: '$TOL'" >&2
            exit 2
            ;;
    esac
    echo "Warning: outline simplification is not supported by the shell version; use Python or Go" >&2
fi

TEMP_CONTENT=$(mktemp)
FINAL_TEMP=$(mktemp)
trap 'rm -f "$TEMP_CONTENT" "$FINAL_TEMP"' EXIT

# 1. Убираем MD5-строку (только первую, как Python/Go) и все старые zmod-строки;
#    нормализуем переводы строк. awk вместо grep -v: grep не умеет "только первую".
awk 'NR == 1 && tolower($0) ~ /^[ \t]*; md5/ { next }
     tolower($0) ~ /^[ \t]*; zmod_color_data =/ { next }
     { print }' "$FILE_PATH" | tr -d '\r' > "$TEMP_CONTENT"

# 2. Номера инструментов в порядке первого появления (как в Python/Go).
#    Разделитель — пробел, таб или ';'; знак допускается, как в Python/Go.
TOOL_IDS=$(awk '{
    l = tolower($0); sub(/^[ \t]+/, "", l)
    split(l, a, /[ \t;]/); tok = a[1]
    if (tok ~ /^t[+-]?[0-9]+$/) { n = substr(tok, 2) + 0; if (!seen[n]++) out = out (out == "" ? "" : ",") n }
} END { print out }' "$TEMP_CONTENT")
[[ -z "$TOOL_IDS" ]] && TOOL_IDS="0"
MAX_TOOL=$(printf '%s\n' "$TOOL_IDS" | tr ',' '\n' | sort -n | tail -n 1)

# 3. Цвета и типы филамента: берём последнюю строку, нижний регистр, ';' -> ','.
#    Пробелы убираем только по краям значения — внутри сохраняются, как в Python/Go.
FILAMENT_COLOURS=$( { grep -aiE '^[[:space:]]*; filament_colour =' "$TEMP_CONTENT" || true; } | tail -n 1 | cut -d'=' -f2- | sed 's/^[[:space:]]*//; s/[[:space:]]*$//' | tr ';' ',' | tr '[:upper:]' '[:lower:]')
FILAMENT_TYPES=$( { grep -aiE '^[[:space:]]*; filament_type =' "$TEMP_CONTENT" || true; } | tail -n 1 | cut -d'=' -f2- | sed 's/^[[:space:]]*//; s/[[:space:]]*$//' | tr ';' ',' | tr '[:upper:]' '[:lower:]')

# Дополняем пустыми полями до числа инструментов (как Python/Go).
pad_fields() {
    printf '%s\n' "$1" | awk -v want="$((MAX_TOOL + 1))" -F',' '{
        have = ($0 == "") ? 0 : NF
        out = $0
        if (have == 0) { for (i = 1; i < want; i++) out = out "," }
        else { for (i = have; i < want; i++) out = out "," }
        printf "%s", out
    }'
}
FILAMENT_COLOURS=$(pad_fields "$FILAMENT_COLOURS")
FILAMENT_TYPES=$(pad_fields "$FILAMENT_TYPES")

ZMOD_LINE="; zmod_color_data = ${TOOL_IDS}|${FILAMENT_COLOURS}|${FILAMENT_TYPES}"

# 4. Вставляем перед первым HEADER_BLOCK_END; если его нет — в начало файла.
#    awk вместо sed 'i': sed 'i text' не работает в BSD/macOS sed.
#    Значение передаём через ENVIRON: awk -v интерпретирует escape-последовательности.
#    Записи выводим без ORS — хвостовой перевод строки сохраняем как в исходнике.
if grep -qaiE '^[[:space:]]*; header_block_end' "$TEMP_CONTENT"; then
    ZMOD_LINE="$ZMOD_LINE" awk '{
        if (NR > 1) printf "\n"
        if (!ins && tolower($0) ~ /^[ \t]*; header_block_end/) { printf "%s\n", ENVIRON["ZMOD_LINE"]; ins = 1 }
        printf "%s", $0
    }' "$TEMP_CONTENT" > "$FINAL_TEMP"
else
    ZMOD_LINE="$ZMOD_LINE" awk 'BEGIN { printf "%s", ENVIRON["ZMOD_LINE"] } { printf "\n%s", $0 }' "$TEMP_CONTENT" > "$FINAL_TEMP"
fi
if [[ -z "$(tail -c 1 "$FILE_PATH" | tr -d '\r\n')" ]]; then
    printf '\n' >> "$FINAL_TEMP"
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
