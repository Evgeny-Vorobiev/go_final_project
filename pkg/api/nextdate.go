package api

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// NextDate вычисляет следующую дату повторения задачи, начиная с dstart,
// с учётом правила repeat. Возвращает дату в формате YYYYMMDD или пустую строку,
// если правило не распознано или задано неверно.
// now — текущее время, используется как порог: следующая дата должна быть строго больше now.
func NextDate(now time.Time, dstart string, repeat string) (string, error) {
	start, err := time.Parse(dateFormat, dstart)
	if err != nil {
		return "", fmt.Errorf("invalid start date %q: %w", dstart, err)
	}

	parts := strings.Fields(repeat)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty repeat rule")
	}

	switch parts[0] {
	case "d":
		return nextDay(start, now, parts)
	case "y":
		return nextYear(start, now, repeat)
	case "w":
		return nextWeekday(start, now, parts)
	case "m":
		return nextMonthDay(start, now, parts)
	default:
		return "", fmt.Errorf("недопустимое правило повторения")
	}
}

// nextDay реализует правило "d N" — повторение каждые N дней.
func nextDay(start, now time.Time, parts []string) (string, error) {
	if len(parts) < 2 {
		return "", fmt.Errorf("rule 'd' requires number of days")
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil || n < 1 || n > 400 {
		return "", fmt.Errorf("invalid days count %q for rule 'd': must be 1–400", parts[1])
	}

	date := start
	for {
		date = date.AddDate(0, 0, n)
		if date.After(now) {
			return date.Format(dateFormat), nil
		}
	}
}

// nextYear реализует правило "y" — ежегодно.
func nextYear(start, now time.Time, repeat string) (string, error) {
	if repeat != "y" {
		return "", nil
	}

	date := start
	for {
		date = date.AddDate(1, 0, 0)
		if date.After(now) {
			return date.Format(dateFormat), nil
		}
	}
}

// nextWeekday реализует правило "w дни" — повторение по дням недели (1-Пн, …, 7-Вс).
func nextWeekday(start, now time.Time, parts []string) (string, error) {
	if len(parts) < 2 {
		return "", fmt.Errorf("rule 'w' requires weekdays list, e.g. 'w 1,3,5'")
	}
	days, err := parseWeekdays(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid weekdays specification: %w", err)
	}

	date := start
	for {
		if date.After(now) {
			wd := int(date.Weekday()) // Go: 0=Вс, 1=Пн, ..., 6=Сб
			if wd == 0 {
				wd = 7 // переводим в 1..7 (1=Пн, 7=Вс)
			}

			if slices.Contains(days, wd) {
				return date.Format(dateFormat), nil
			}

		}
		date = date.AddDate(0, 0, 1)
		// Ограничение на случай бесконечного цикла (если ни один день не совпадёт)
		if date.Year() > start.Year()+2 {
			break
		}
	}
	return "", nil
}

// parseWeekdays преобразует строку вида "1,3,5" в слайс целых чисел и проверяет корректность.
func parseWeekdays(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	res := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n < 1 || n > 7 {
			return nil, fmt.Errorf("invalid weekday %q: must be 1–7", p)
		}
		res = append(res, n)
	}
	return res, nil
}

// nextMonthDay реализует правило "m дни [месяцы]" — повторение в указанные дни месяца,
// опционально с фильтром по месяцам. Поддерживаются отрицательные значения (-1 = последний день, -2 = предпоследний).
func nextMonthDay(start, now time.Time, parts []string) (string, error) {
	if len(parts) < 2 {
		return "", fmt.Errorf("rule 'm' requires at least days list")
	}

	days, months, err := parseMonthArgs(parts[1:])
	if err != nil {
		return "", fmt.Errorf("invalid month/day arguments: %w", err)
	}

	date := start
	for {
		if date.After(now) {
			if matchesMonthDay(date, days, months) {
				return date.Format(dateFormat), nil
			}
		}
		date = date.AddDate(0, 0, 1)
		if date.Year() > start.Year()+2 {
			break
		}
	}
	return "", nil
}

// parseMonthArgs разбирает аргументы правила m: первый элемент — дни (через запятую),
// второй опционально — месяцы (через запятую).
func parseMonthArgs(parts []string) (days []int, months []int, err error) {
	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("no args")
	}

	// Дни
	dayStrs := strings.Split(parts[0], ",")
	for _, s := range dayStrs {
		n, e := strconv.Atoi(strings.TrimSpace(s))
		if e != nil {
			return nil, nil, fmt.Errorf("invalid day %q: %w", s, e)
		}
		// Валидация: допускаются только 1..31 и -1, -2
		if n < -2 || n > 31 || n == 0 {
			return nil, nil, fmt.Errorf("invalid day value %d: must be in [-2,31] excluding 0", n)
		}
		days = append(days, n)
	}

	// Месяцы (если есть)
	if len(parts) > 1 {
		monthStrs := strings.Split(parts[1], ",")
		months = make([]int, 0, len(monthStrs))
		for _, s := range monthStrs {
			n, e := strconv.Atoi(strings.TrimSpace(s))
			if e != nil || n < 1 || n > 12 {
				return nil, nil, fmt.Errorf("invalid month: %s", s)
			}
			months = append(months, n)
		}
	}

	return days, months, nil
}

// matchesMonthDay проверяет, соответствует ли дата date заданным дням и месяцам.
func matchesMonthDay(date time.Time, days []int, months []int) bool {
	if months != nil {
		foundMonth := false
		for _, m := range months {
			if int(date.Month()) == m {
				foundMonth = true
				break
			}
		}
		if !foundMonth {
			return false
		}
	}

	day := date.Day()
	daysInMonth := time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()

	for _, d := range days {
		target := d
		if d < 0 {
			// -1 -> последний день, -2 -> предпоследний
			target = daysInMonth + d + 1
		}
		if target == day {
			return true
		}
	}
	return false
}

// handleNextDate обрабатывает GET /api/nextdate?now=...&date=...&repeat=...
// Вычисляет следующую дату повторения и возвращает её в теле ответа.
// Если now не задан, используется текущее системное время.
func handleNextDate(w http.ResponseWriter, r *http.Request) {
	nowStr := r.URL.Query().Get("now")
	dateStr := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")

	var now time.Time
	var err error

	if nowStr == "" {
		now = time.Now()
	} else {
		now, err = time.Parse(dateFormat, nowStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "неверный формат параметра now", err)
			return
		}
	}

	next, err := NextDate(now, dateStr, repeat)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ошибка вычисления даты", err)
		return
	}

	if next == "" {
		writeError(w, http.StatusBadRequest, "не удалось вычислить следующую дату", nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(next))
}
