package validator

import (
	"strconv"
	"strings"
)

func ValidateLuhn(number string) bool {
	// 1. Удаляем пробелы, если они есть
	number = strings.ReplaceAll(number, " ", "")

	var sum int
	var alternate bool

	// 2. Идем с конца строки
	for i := len(number) - 1; i >= 0; i-- {
		n, err := strconv.Atoi(string(number[i]))
		if err != nil {
			return false // Нецифровой символ
		}

		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}

		sum += n
		alternate = !alternate
	}

	// 3. Сумма должна делиться на 10
	return sum%10 == 0
}
