package lib

// SwitchEndian convert a big endian to a little endian hex representation
func SwitchEndian(input string) (result string) {
	for i := 0; i < len(input); i = i + 2 {
		result = input[i:i+2] + result
	}

	return result
}
