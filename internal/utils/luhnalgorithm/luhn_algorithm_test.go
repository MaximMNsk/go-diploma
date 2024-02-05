package luhnalgorithm

import "testing"

func TestIsLuhnValid(t *testing.T) {
	//data := []int{7, 9, 9, 2, 7, 3, 9, 8, 7, 1, 3}
	data := "79927398713"
	got, _ := IsLuhnValid(data)
	if !got {
		t.Errorf("Expected '%v' is Luhn valid", data)
	}
}

func TestLuhnInvalid(t *testing.T) {
	//data := []int{7, 9, 9, 2, 7, 3, 9, 8, 7, 1}
	data := "79927398711"
	got, _ := IsLuhnValid(data)
	if got {
		t.Errorf("Expected '%v' is not Luhn valid", data)
	}
}

func TestLuhn(t *testing.T) {
	data := []int{7, 9, 9, 2, 7, 3, 9, 8, 7, 1}
	expected := 3
	got := Luhn(data)
	if got != expected {
		t.Errorf("Expected %d got %d", expected, got)
	}
}

func TestLuhns(t *testing.T) {
	data := "7992739871"
	expected := 3
	got, err := Luhns(data)
	if err != nil {
		t.Errorf("Failed to calculate the Luhn Algorithm check digit to %s. err: %v", data, err)
	}
	if got != expected {
		t.Errorf("Expected %d got %d", expected, got)
	}
}

func TestLuhnsWithError(t *testing.T) {
	data := "799273987X"
	_, err := Luhns(data)
	if err == nil {
		t.Errorf("Should not be possible to get a number representation of '%s'", data)
	}

}

func TestLuhni(t *testing.T) {
	data := 7992739871
	expected := 3
	got := Luhni(data)
	if got != expected {
		t.Errorf("Expected %d got %d", expected, got)
	}
}
