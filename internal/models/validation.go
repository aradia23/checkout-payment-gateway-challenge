package models

import (
	"fmt"
	"time"
)

type currencyCode string

const (
	minLengthCardNumber = 14
	maxLengthCardNumber = 19

	minMonth = 1
	maxMonth = 12
)

var supportedCurrencies = map[string]bool{
	"USD": true,
	"EUR": true,
	"GBP": true,
}

func (r PostPaymentRequest) Validate() []error {
	var errs []error
	err := validateCardNumber(r.CardNumber)
	if err != nil {
		errs = append(errs, err)
	}

	err = validExpiryDate(r.ExpiryMonth, r.ExpiryYear)
	if err != nil {
		errs = append(errs, err)
	}

	err = validateCurrencyCode(r.Currency)
	if err != nil {
		errs = append(errs, err)
	}

	err = validateAmount(r.Amount)
	if err != nil {
		errs = append(errs, err)
	}

	return errs
}

func validateCardNumber(cardNumber string) error {
	if !isNumeric(cardNumber) {
		return fmt.Errorf("card number must only contain digits between 0-9, %s", cardNumber)
	}

	if len(cardNumber) < minLengthCardNumber || len(cardNumber) > maxLengthCardNumber {
		return fmt.Errorf("invalid card number lenth: %s", cardNumber)
	}

	return nil
}

func validateMonth(month int) error {
	if month < minMonth || month > maxMonth {
		return fmt.Errorf("invalid expiry month, it must be between 1-12 %d", month)
	}
	return nil
}

func validateYear(year int) error {
	// Assuming year is in YYYY format
	if year < 1000 || year > 9999 {
		return fmt.Errorf("invalid year format, it must be in YYYY format %d", year)
	}
	return nil
}

func validExpiryDate(month int, year int) error {
	var err error
	err = validateMonth(month)
	if err != nil {
		return err
	}

	err = validateYear(year)
	if err != nil {
		return err
	}

	now := time.Now()

	expiry := time.Date(
		year,
		time.Month(month)+1,
		1,
		0, 0, 0, 0,
		time.Local,
	)

	if expiry.Before(now) {
		return fmt.Errorf("expiry date must not be before current date")
	}

	return nil
}

func validateCurrencyCode(code string) error {
	if !supportedCurrencies[string(code)] {
		return fmt.Errorf("unsupported currency code: %s", code)
	}
	return nil
}

func validateAmount(amount int) error {
	if amount < 0 {
		return fmt.Errorf("amount must be a positive integer: %d", amount)
	}
	return nil
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
