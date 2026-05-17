package validator

import (
	"strconv"
	"strings"
)

func ValidateLuhn(number string) bool {
	number = strings.ReplaceAll(number, " ", "")

	var sum int
	var alternate bool
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
	return sum%10 == 0
}
