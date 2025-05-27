package shared

import "testing"

func TestGetPubkeyFromNpub(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		err      bool
	}{
		{"npub15qydau2hjma6ngxkl2cyar74wzyjshvl65za5k5rl69264ar2exs5cyejr", "a008def15796fba9a0d6fab04e8fd57089285d9fd505da5a83fe8aad57a3564d", false},
		{"naddr1qvzqqqrhnypzpgqgmmc409hm4xsdd74sf68a2uyf9pwel4g9mfdg8l5244t6x4jdqy28wumn8ghj7un9d3shjtnyv9kh2uewd9hsqzm8d968wmmjddeksmmsu4xedt", "", true},
		{"invalid_format", "", true},
	}

	for _, test := range tests {
		result, err := GetPubkeyFromNpub(test.input)
		if (err != nil) != test.err {
			t.Errorf("GetPubkeyFromNpub(%s) error = %v, wantErr %v", test.input, err, test.err)
			continue
		}
		if result != test.expected {
			t.Errorf("GetPubkeyFromNpub(%s) = %v, want %v", test.input, result, test.expected)
		}
	}
}
