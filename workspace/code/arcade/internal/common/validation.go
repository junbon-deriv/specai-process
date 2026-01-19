package common

import (
	"regexp"
)

var (
	// Currency code must be exactly 3 uppercase letters
	currencyRegex = regexp.MustCompile(`^[A-Z]{3}$`)

	// Account ID format: SW followed by digits
	accountIDRegex = regexp.MustCompile(`^SW\d+$`)
)

// ValidateCurrency validates currency code format (3 uppercase letters)
func ValidateCurrency(currency string) error {
	if !currencyRegex.MatchString(currency) {
		return ErrInvalidCurrency
	}
	return nil
}

// ValidateAccountID validates account ID format (SW prefix + digits)
func ValidateAccountID(accountID string) bool {
	return accountIDRegex.MatchString(accountID)
}

// ValidateSeriesType validates series type
func ValidateSeriesType(seriesType string) error {
	switch seriesType {
	case "Vol50", "Vol100", "Vol200", "Vol300":
		return nil
	default:
		return ErrInvalidSeriesType
	}
}

// ValidateSentiment validates sentiment
func ValidateSentiment(sentiment string) error {
	switch sentiment {
	case "rise", "fall":
		return nil
	default:
		return ErrInvalidSentiment
	}
}
