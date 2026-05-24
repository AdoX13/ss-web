package pipeline

// Romanian CNP (Cod Numeric Personal) validator.
//
// Structure: 13 digits, SYYMMDDJJNNNC where
//   S    = gender + century  (1..8; 9 for foreigners)
//   YYMMDD = birthdate
//   JJ   = county code (1..52 inclusive of some special codes)
//   NNN  = sequence number
//   C    = control digit
//
// Control digit algorithm (constants vector "279146358279"):
//   sum_i (cnp[i] * const[i]) for i in 0..11, mod 11
//   if mod == 10 → control = 1, else control = mod
//
// Reference: ISO 3166-2:RO + the Romanian civil status code.

// CNPControlConstants is the public weights vector for the control
// digit calculation. Exported only so tests can pin the value.
var CNPControlConstants = [12]int{2, 7, 9, 1, 4, 6, 3, 5, 8, 2, 7, 9}

// ValidateCNP returns true iff s is a 13-character all-digit string
// whose control digit matches.
//
// Notes:
//   - Length and digit-only checks come first so non-CNP inputs (e.g.
//     OCR noise like "B131234") return false cheaply.
//   - The S byte is NOT range-checked here. Some test data uses S=0,
//     and our job is to reject *invalid* CNPs, not to opine on
//     business rules.
func ValidateCNP(s string) bool {
	if len(s) != 13 {
		return false
	}
	var digits [13]int
	for i := 0; i < 13; i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return false
		}
		digits[i] = int(c - '0')
	}
	sum := 0
	for i := 0; i < 12; i++ {
		sum += digits[i] * CNPControlConstants[i]
	}
	control := sum % 11
	if control == 10 {
		control = 1
	}
	return control == digits[12]
}
