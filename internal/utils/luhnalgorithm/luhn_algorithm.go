package luhnalgorithm

import (
	"errors"
	"strconv"
)

// IsLuhnValid validates a number according to the Luhn formula
func IsLuhnValid(value string) (bool, error) {
	var err error
	var valueSlice []int

	if len(value) < 2 {
		return false, errors.New(`value too small`)
	}

	for i := 0; i < len(value); i++ {
		digit, err := strconv.Atoi(value[i : i+1])
		if err != nil {
			return false, err
		}
		valueSlice = append(valueSlice, digit)
	}

	sum := computeCheckSum(valueSlice)
	return sum%10 == 0, err
}

// Luhn calculate the Luhn Algorithm check digit for a slice ([]int)
func Luhn(value []int) int {
	sum := computeCheckSum(value)
	return computeCheckDigit(sum)
}

// Luhns calculate the Luhn Algorithm check digit for a string
func Luhns(value string) (int, error) {
	var data []int
	for _, s := range value {
		n, err := strconv.Atoi(string(s))
		if err != nil {
			return -1, err
		}
		data = append(data, n)
	}
	return Luhn(data), nil
}

// Luhni calculate the Luhn Algorithm check digit for a number
func Luhni(value int) int {
	s := strconv.Itoa(value)
	l, _ := Luhns(s)
	return l
}

func computeCheckSum(data []int) int {
	var sum int
	double := false
	for _, n := range data {
		if double {
			n = (n * 2)
			if n > 9 {
				n = (n - 9)
			}
		}
		sum += n
		double = !double
	}
	return sum
}

// The computeCheckDigit check digit is obtained by computing the sum of the other digits
// then subtracting the units digit from 10. In algorithm form:
// * Compute the sum of the digits (e.g., 67).
// * Take the units digit (7).
// * Subtract the units digit from 10. (10 - 7 = 3)
// The result (3) is the check digit. In case the sum of digits ends in 0, 0 is the check digit.
func computeCheckDigit(sum int) int {
	unitDigit := sum % 10
	return 10 - unitDigit
}
