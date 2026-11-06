package ui

import "testing"

func TestValidatePort(t *testing.T) {
	for in, ok := range map[string]bool{
		"": true, "  ": true, "1": true, " 22 ": true, "65535": true,
		"0": false, "65536": false, "-1": false, "x": false, "22x": false,
	} {
		if err := validatePort(in); (err == nil) != ok {
			t.Errorf("validatePort(%q) err=%v, want ok=%v", in, err, ok)
		}
	}
}

func TestValidateNonNegativeInt(t *testing.T) {
	for in, ok := range map[string]bool{
		"": true, "  ": true, "0": true, " 7 ": true,
		"-1": false, "x": false, "1.5": false,
	} {
		if err := validateNonNegativeInt(in); (err == nil) != ok {
			t.Errorf("validateNonNegativeInt(%q) err=%v, want ok=%v", in, err, ok)
		}
	}
}
