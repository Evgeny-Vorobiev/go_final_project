package api

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

const dateFormat = "20060102"

func NextDate(now time.Time, dstart string, repeat string) (string, error) {
	if repeat == "" {
		return "", errors.New("repeat rule is empty")
	}

	start, err := time.Parse(dateFormat, dstart)
	if err != nil {
		return "", err
	}

	parts := strings.Split(strings.TrimSpace(repeat), " ")
	if len(parts) == 0 {
		return "", errors.New("invalid repeat format")
	}

	rule := parts[0]

	// Базовые правила
	if rule == "d" {
		if len(parts) != 2 {
			return "", errors.New("invalid d rule: expected 'd <days>'")
		}
		days, err := strconv.Atoi(parts[1])
		if err != nil || days <= 0 || days > 400 {
			return "", errors.New("invalid days value for rule d")
		}
		date := start
		for {
			date = date.AddDate(0, 0, days)
			if date.After(now) {
				return date.Format(dateFormat), nil
			}
		}
	} else if rule == "y" {
		date := start
		for {
			date = date.AddDate(1, 0, 0)
			if date.After(now) {
				return date.Format(dateFormat), nil
			}
		}
	}

	// Правила со звёздочкой
	if rule == "w" {
		if len(parts) != 2 {
			return "", errors.New("invalid w rule: expected 'w <days>'")
		}
		ws := strings.Split(parts[1], ",")
		validDays := make([]int, 0, len(ws))
		for _, w := range ws {
			d, err := strconv.Atoi(w)
			if err != nil || d < 1 || d > 7 {
				return "", errors.New("invalid weekday in rule w")
			}
			validDays = append(validDays, d)
		}

		date := start.AddDate(0, 0, 1)
		for {
			wd := int(date.Weekday())
			if wd == 0 { // Sunday
				wd = 7
			}
			for _, v := range validDays {
				if v == wd {
					if date.After(now) {
						return date.Format(dateFormat), nil
					}
				}
			}
			date = date.AddDate(0, 0, 1)
		}
	} else if rule == "m" {
		if len(parts) < 2 || len(parts) > 3 {
			return "", errors.New("invalid m rule: expected 'm <days>[, <months>]'")
		}

		parseDays := func(s string) ([]int, error) {
			ds := strings.Split(s, ",")
			res := make([]int, 0, len(ds))
			for _, d := range ds {
				val, err := strconv.Atoi(d)
				if err != nil || val == 0 || val < -31 || val > 31 {
					return nil, errors.New("invalid day value in rule m")
				}
				res = append(res, val)
			}
			return res, nil
		}

		days, err := parseDays(parts[1])
		if err != nil {
			return "", err
		}

		months := []int{}
		if len(parts) == 3 {
			ms := strings.Split(parts[2], ",")
			for _, m := range ms {
				mm, err := strconv.Atoi(m)
				if err != nil || mm < 1 || mm > 12 {
					return "", errors.New("invalid month value in rule m")
				}
				months = append(months, mm)
			}
		}

		date := start.AddDate(0, 0, 1)
		for {
			y, m, d := date.Date()

			okDay := false
			for _, dayVal := range days {
				if dayVal > 0 && dayVal == d {
					okDay = true
					break
				}
				if dayVal < 0 {
					last := time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1).Day()
					// dayVal = -1 -> последний день, -2 -> предпоследний и т.д.
					if last+dayVal+1 == d {
						okDay = true
						break
					}
				}
			}
			if !okDay {
				date = date.AddDate(0, 0, 1)
				continue
			}

			okMonth := len(months) == 0
			if len(months) > 0 {
				for _, mm := range months {
					if int(m) == mm {
						okMonth = true
						break
					}
				}
			}
			if okMonth {
				if date.After(now) {
					return date.Format(dateFormat), nil
				}
			}
			date = date.AddDate(0, 0, 1)
		}
	}

	return "", errors.New("unsupported repeat rule")
}
