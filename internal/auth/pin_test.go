package auth

import "testing"

func TestValidatePINFormat(t *testing.T) {
	cases := map[string]bool{
		"1234":    true,
		"123456":  true,
		"12345":   true,
		"123":     false,
		"1234567": false,
		"12a4":    false,
		"":        false,
		"abcd":    false,
	}
	for pin, want := range cases {
		err := ValidatePINFormat(pin)
		if got := err == nil; got != want {
			t.Errorf("ValidatePINFormat(%q) valid=%v, want %v (err=%v)", pin, got, want, err)
		}
	}
}

func TestHashPINThenVerify(t *testing.T) {
	hash, salt, err := HashPIN("4321")
	if err != nil {
		t.Fatalf("HashPIN: %v", err)
	}
	if hash == "" || salt == "" {
		t.Fatalf("expected non-empty hash and salt")
	}
	if !VerifyPIN("4321", hash, salt) {
		t.Fatal("VerifyPIN should accept the correct PIN")
	}
	if VerifyPIN("4322", hash, salt) {
		t.Fatal("VerifyPIN should reject a wrong PIN")
	}
}

func TestHashPINProducesDistinctSalts(t *testing.T) {
	_, salt1, err := HashPIN("1111")
	if err != nil {
		t.Fatalf("HashPIN: %v", err)
	}
	_, salt2, err := HashPIN("1111")
	if err != nil {
		t.Fatalf("HashPIN: %v", err)
	}
	if salt1 == salt2 {
		t.Fatal("expected two hashes of the same PIN to use different salts")
	}
}

func TestVerifyPINRejectsGarbageEncoding(t *testing.T) {
	if VerifyPIN("1234", "not-base64!!", "also-not-base64!!") {
		t.Fatal("VerifyPIN should reject undecodable hash/salt rather than panic")
	}
}
