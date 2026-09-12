package models

import "testing"

func TestValidateCardNumber(t *testing.T) {
	tests := []struct {
		name       string
		cardNumber string
		wantErr    bool
	}{
		{name: "minimum length", cardNumber: "12345678901234"},
		{name: "maximum length", cardNumber: "1234567890123456789"},
		{name: "too short", cardNumber: "1234567890123", wantErr: true},
		{name: "too long", cardNumber: "12345678901234567890", wantErr: true},
		{name: "contains non-digit", cardNumber: "1234567890123a", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateCardNumber(test.cardNumber); (err != nil) != test.wantErr {
				t.Fatalf("validateCardNumber() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestValidExpiryDate(t *testing.T) {
	if err := validExpiryDate(1, 2000); err != nil {
		t.Fatalf("validExpiryDate() error for an expired date = %v", err)
	}

	if err := validExpiryDate(12, 9999); err == nil {
		t.Error("validExpiryDate() expected an error for a future date")
	}

	if err := validExpiryDate(13, 2030); err == nil {
		t.Error("validExpiryDate() expected an error for an invalid month")
	}

	if err := validExpiryDate(1, 999); err == nil {
		t.Error("validExpiryDate() expected an error for an invalid year")
	}
}

func TestValidateCurrencyCode(t *testing.T) {
	if err := validateCurrencyCode("USD"); err != nil {
		t.Fatalf("validateCurrencyCode(USD) error = %v", err)
	}

	if err := validateCurrencyCode("JPY"); err == nil {
		t.Error("validateCurrencyCode(JPY) expected an error for unsupported currency")
	}
}

func TestValidateAmount(t *testing.T) {
	if err := validateAmount(0); err != nil {
		t.Fatalf("validateAmount(0) error = %v", err)
	}

	if err := validateAmount(100); err != nil {
		t.Fatalf("validateAmount(100) error = %v", err)
	}

	if err := validateAmount(-1); err == nil {
		t.Error("validateAmount(-1) expected an error")
	}
}
