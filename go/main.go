// (C) Namida Verasche aka ninjamida
// (C) ghzserg https://zmod.link
// MD5 aspects copied (with changes) from addMD5.py

package main

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
)

func main() {
	// Путь к файлу остаётся позиционным, ключи необязательные. Слайсер
	// дописывает путь последним, поэтому ключи всегда идут до него.
	toleranceFlag := flag.Float64("simplify-objects", 0.2,
		"outline tolerance in mm; 0 leaves the bounding rectangle")
	noSimplify := flag.Bool("no-simplify-objects", false, "keep the full outline")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: addColorAndMD5 [-simplify-objects=tolerance_mm] [-no-simplify-objects] <file_path>")
		flag.PrintDefaults()
	}
	flag.Parse()

	tolerance := *toleranceFlag
	if tolerance < 0 {
		fmt.Fprintf(os.Stderr, "Bad tolerance: %v\n", tolerance)
		os.Exit(1)
	}
	simplify := !*noSimplify
	filePath := flag.Arg(0)
	if filePath == "" {
		flag.Usage()
		os.Exit(1)
	}

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
		if simplify && strings.Contains(lineRaw, "EXCLUDE_OBJECT_DEFINE") {
			if simplified, ok := simplifyPolygon(lineRaw, tolerance); ok {
				lines[i] = simplified
				lineRaw = simplified
			}
		}

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
		    // Делим строку на части до комментария
		    parts := strings.SplitN(line, ";", 2)
		    mainPart := parts[0]

		    // Делим на части до пробела или других символов
		    mainParts := strings.SplitN(mainPart, " ", 2)
		    toolCommand := strings.TrimSpace(mainParts[0])

		    // Проверяем, что это действительно команда T с числом
		    if len(toolCommand) > 1 && toolCommand[0] == 't' {
		        if idx, err := strconv.Atoi(toolCommand[1:]); err == nil {
		            resultColors[idx] = true
		            if idx > highestResultColor {
		                highestResultColor = idx
		            }
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

type point struct{ x, y float64 }

// simplifyPolygon прореживает контур детали в EXCLUDE_OBJECT_DEFINE. Слайсер пишет
// контур первого слоя целиком, и на плите из полусотни мелких деталей набирается
// больше десяти тысяч точек — их разбор подвешивает Klipper. Отмена детали идёт по
// имени и центру, а KAMP и LINE_PURGE берут из полигона только габарит, поэтому он
// сохраняется точно при любом допуске.
func simplifyPolygon(line string, tolerance float64) (string, bool) {
	start := strings.Index(line, "POLYGON=")
	if start < 0 {
		return line, false
	}
	start += len("POLYGON=")

	// Значение — обычный JSON, а декодер сам сообщает, где оно кончилось: дальше
	// в строке идёт остаток команды. Всё, что не разобралось, оставляем как есть:
	// нетронутый контур лучше собранного из неверно прочитанных точек.
	dec := json.NewDecoder(strings.NewReader(line[start:]))
	var raw [][]float64
	if err := dec.Decode(&raw); err != nil || len(raw) < 5 {
		return line, false
	}
	end := start + int(dec.InputOffset())

	pts := make([]point, 0, len(raw))
	for _, pair := range raw {
		if len(pair) != 2 {
			return line, false
		}
		pts = append(pts, point{pair[0], pair[1]})
	}

	simple := simplifyRing(pts, tolerance)
	if len(simple) >= len(pts) {
		return line, false
	}

	var b strings.Builder
	b.WriteByte('[')
	for i, p := range simple {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "[%s,%s]",
			strconv.FormatFloat(p.x, 'f', -1, 64), strconv.FormatFloat(p.y, 'f', -1, 64))
	}
	b.WriteByte(']')
	return line[:start] + b.String() + line[end:], true
}

// simplifyRing прореживает замкнутый контур методом Дугласа-Пекера. Крайние по осям
// точки объявлены неудаляемыми, поэтому габарит не меняется. Нулевой допуск даёт
// сам габаритный прямоугольник.
func simplifyRing(pts []point, tolerance float64) []point {
	minX, maxX, minY, maxY := 0, 0, 0, 0
	for i, p := range pts {
		if p.x < pts[minX].x {
			minX = i
		}
		if p.x > pts[maxX].x {
			maxX = i
		}
		if p.y < pts[minY].y {
			minY = i
		}
		if p.y > pts[maxY].y {
			maxY = i
		}
	}
	if tolerance <= 0 {
		x0, y0, x1, y1 := pts[minX].x, pts[minY].y, pts[maxX].x, pts[maxY].y
		return []point{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
	}

	anchors := []int{minX, maxX, minY, maxY}
	slices.Sort(anchors)
	anchors = slices.Compact(anchors)
	if len(anchors) < 2 {
		return pts
	}

	var out []point
	for i, a := range anchors {
		b := anchors[(i+1)%len(anchors)]
		var arc []point
		if b > a {
			arc = pts[a : b+1]
		} else {
			arc = append(slices.Clone(pts[a:]), pts[:b+1]...) // последняя дуга через начало
		}
		kept := douglasPeucker(arc, tolerance)
		out = append(out, kept[:len(kept)-1]...) // конец дуги — начало следующей
	}
	return out
}

// douglasPeucker оставляет из дуги те точки, что отходят от хорды дальше допуска.
// Концы дуги сохраняются всегда.
func douglasPeucker(pts []point, tolerance float64) []point {
	if len(pts) < 3 {
		return slices.Clone(pts)
	}
	a, b := pts[0], pts[len(pts)-1]
	dx, dy := b.x-a.x, b.y-a.y
	chord := math.Hypot(dx, dy)
	dist := func(p point) float64 {
		if chord == 0 {
			return math.Hypot(p.x-a.x, p.y-a.y)
		}
		return math.Abs(dx*(a.y-p.y)-dy*(a.x-p.x)) / chord
	}

	worst, at := 0.0, 0
	for i := 1; i < len(pts)-1; i++ {
		if d := dist(pts[i]); d > worst {
			worst, at = d, i
		}
	}
	if worst <= tolerance {
		return []point{a, b}
	}
	head := douglasPeucker(pts[:at+1], tolerance)
	return append(head[:len(head)-1], douglasPeucker(pts[at:], tolerance)...)
}
