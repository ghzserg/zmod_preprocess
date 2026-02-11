// (C) Namida Verasche aka ninjamida
// (C) ghzserg https://zmod.link
// MD5 aspects copied (with changes) from addMD5.py

package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: addColorAndMD5 <file_path>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	// Чтение файла построчно
	var lines []string
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Удаляем первую строку, если она содержит '; MD5' (case-insensitive)
	if len(lines) > 0 && strings.HasPrefix(strings.ToLower(strings.TrimSpace(lines[0])), "; md5") {
		lines = lines[1:]
	}

	// Переменные для сбора данных
	resultColors := make(map[int]bool)
	highestResultColor := -1
	var filamentColorLine, filamentTypeLine string
	headerEndLine := -1
	removeExistingLine := -1
	foundHeaderEndLine := false
	foundExistingLine := false

	// Парсим содержимое
	for i, lineRaw := range lines {
		line := strings.ToLower(strings.TrimSpace(lineRaw))

		if !foundHeaderEndLine {
			headerEndLine = i
		}
		if !foundExistingLine {
			removeExistingLine = i
		}

		if line == "" {
			continue
		}

		// Поиск команд вида T0, T1 и т.д.
		if strings.HasPrefix(line, "t") {
			if idx, err := strconv.Atoi(line[1:]); err == nil {
				resultColors[idx] = true
				if idx > highestResultColor {
					highestResultColor = idx
				}
			}
		}

		// Извлечение данных о цвете и типе филамента
		if strings.HasPrefix(line, "; filament_colour =") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) > 1 {
				filamentColorLine = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "; filament_type =") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) > 1 {
				filamentTypeLine = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "; zmod_color_data =") {
			foundExistingLine = true
		} else if strings.HasPrefix(line, "; header_block_end") {
			foundHeaderEndLine = true
		}
	}

	// Формируем список уникальных индексов цветов
	var colorIndexes []int
	for idx := range resultColors {
		colorIndexes = append(colorIndexes, idx)
	}
	if len(colorIndexes) == 0 {
		colorIndexes = []int{0}
		highestResultColor = 0
	}

	// Парсим цвета и типы филамента
	filamentColors := strings.Split(filamentColorLine, ";")
	filamentTypes := strings.Split(filamentTypeLine, ";")

	if len(filamentColors) > 0 && filamentColors[0] == "" {
		filamentColors = []string{}
	}
	if len(filamentTypes) > 0 && filamentTypes[0] == "" {
		filamentTypes = []string{}
	}

	// Дополняем массивы до нужной длины
	for len(filamentColors) <= highestResultColor {
		filamentColors = append(filamentColors, "")
	}
	for len(filamentTypes) <= highestResultColor {
		filamentTypes = append(filamentTypes, "")
	}

	// Формируем строку данных БЕЗ завершающего \r\n
	toolIndexesStr := make([]string, len(colorIndexes))
	for i, idx := range colorIndexes {
		toolIndexesStr[i] = strconv.Itoa(idx)
	}
	filamentColorStr := strings.Join(filamentColors, ",")
	filamentTypeStr := strings.Join(filamentTypes, ",")

	colorDataLine := fmt.Sprintf("; zmod_color_data = %s|%s|%s",
		strings.Join(toolIndexesStr, ","),
		filamentColorStr,
		filamentTypeStr)

	// Вставка/замена строки с цветовыми данными
	if !foundHeaderEndLine {
		headerEndLine = 0
	}

	// Удаляем существующую строку, если найдена
	if foundExistingLine {
		if removeExistingLine > headerEndLine {
			removeExistingLine++
		}
		lines = append(lines[:removeExistingLine], lines[removeExistingLine+1:]...)
	}

	// Вставляем новую строку перед header_block_end
	lines = append(lines[:headerEndLine], append([]string{colorDataLine}, lines[headerEndLine:]...)...)

	// Формируем окончательное содержимое с \r\n как разделителем (стандарт для G-code)
	finalContent := strings.Join(lines, "\n") + "\n"
	finalContentBytes := []byte(finalContent)

	// Вычисляем MD5
	hash := md5.Sum(finalContentBytes)
	md5Line := fmt.Sprintf("; MD5:%x\n", hash)

	// Записываем результат
	output := md5Line + finalContent
	if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}
}
