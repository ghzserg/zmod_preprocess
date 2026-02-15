#!/bin/bash

FILE_PATH="$1"

if [[ -z "$FILE_PATH" || ! -f "$FILE_PATH" ]]; then
    echo "Usage: $0 <gcode_file>"
    exit 1
fi

TEMP_CONTENT=$(mktemp)
FINAL_TEMP=$(mktemp)

# 1. Remove existing MD5/zmod lines and strip CR for processing
grep -viE '^; (MD5:|zmod_color_data =)' "$FILE_PATH" | tr -d '\r' > "$TEMP_CONTENT"

# 2. Extract Tool Numbers (T0, T1, etc.)
TOOL_IDS=$(grep -iE '^T[0-9]+' "$TEMP_CONTENT" | sed -E 's/^[Tt]([0-9]+).*/\1/' | sort -nu | paste -sd "," -)
[[ -z "$TOOL_IDS" ]] && TOOL_IDS="0"

# 3. Extract Filament Data (stripping spaces and formatting as comma-separated)
FILAMENT_COLOURS=$(grep -i "^; filament_colour =" "$TEMP_CONTENT" | cut -d'=' -f2- | tr -d ' ' | tr ';' ',')
FILAMENT_TYPES=$(grep -i "^; filament_type =" "$TEMP_CONTENT" | cut -d'=' -f2- | tr -d ' ' | tr ';' ',')

# 4. Construct the zmod_color_data line
ZMOD_LINE="; zmod_color_data = ${TOOL_IDS}|${FILAMENT_COLOURS}|${FILAMENT_TYPES}"

# 5. Insert the zmod line
if grep -qi "; HEADER_BLOCK_END" "$TEMP_CONTENT"; then
    sed "/; HEADER_BLOCK_END/i $ZMOD_LINE" "$TEMP_CONTENT" > "$FINAL_TEMP"
else
    cp "$TEMP_CONTENT" "$FINAL_TEMP"
    echo "$ZMOD_LINE" >> "$FINAL_TEMP"
fi

# 6. Calculate MD5
# Note: We calculate on the clean file before adding the MD5 header
if command -v md5sum >/dev/null; then
    MD5_HASH=$(md5sum "$FINAL_TEMP" | awk '{print $1}')
else
    MD5_HASH=$(md5 -q "$FINAL_TEMP")
fi

# 7. Final Assembly with CRLF line endings
# We build the body FIRST, convert to CRLF, THEN hash it.
{
    if grep -qi "; HEADER_BLOCK_END" "$TEMP_CONTENT"; then
        sed "/; HEADER_BLOCK_END/i $ZMOD_LINE" "$TEMP_CONTENT"
    else
        cat "$TEMP_CONTENT"
        echo "$ZMOD_LINE"
    fi
} | sed 's/$/\r/' > "$FINAL_TEMP"

# 8. Calculate MD5 of the actual FINAL bytes
if command -v md5sum >/dev/null; then
    MD5_HASH=$(md5sum "$FINAL_TEMP" | awk '{print $1}')
else
    MD5_HASH=$(md5 -q "$FINAL_TEMP")
fi

# 9. Prepend the MD5 and write to the original file
# We add \r to the MD5 line too to keep the whole file consistent
echo -e "; MD5:${MD5_HASH}\r" > "$FILE_PATH"
cat "$FINAL_TEMP" >> "$FILE_PATH"

# Cleanup
rm "$TEMP_CONTENT" "$FINAL_TEMP"