// Package luhnalgorithm is an implementation of Luhn Algorithm, also known as the Modulus 10
package luhnalgorithm

import "strconv"

// IsLuhnValid validates a number according to the Luhn formula
func IsLuhnValid(value []int) bool {

	// The formula verifies a number against its included check digit,
	// which is usually appended to a partial account number to generate the full account number.
	// This number must pass the following test:
	// * From the rightmost digit, which is the check digit, moving left,
	// double the value of every second digit;
	// if the product of this doubling operation is greater than 9 (e.g., 8 × 2 = 16),
	// then sum the digits of the products (e.g., 16: 1 + 6 = 7, 18: 1 + 8 = 9).
	// * Take the sum of all the digits.
	// * If the total modulo 10 is equal to 0 (if the total ends in zero) then the number is valid according to the Luhn formula; else it is not valid.
	sum := computeCheckSum(value)
	return sum%10 == 0
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

func computeCheckDigit(sum int) int {
	// The check digit is obtained by computing the sum of the other digits
	// then subtracting the units digit from 10. In algorithm form:
	// * Compute the sum of the digits (e.g., 67).
	// * Take the units digit (7).
	// * Subtract the units digit from 10. (10 - 7 = 3)
	// The result (3) is the check digit. In case the sum of digits ends in 0, 0 is the check digit.
	unitDigit := sum % 10
	return 10 - unitDigit
}
