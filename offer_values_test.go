package mintegral

import "testing"

func TestBudgetAmount_accepts_open_and_decimal_string(t *testing.T) {
	// Given
	var open BudgetAmount
	var decimal BudgetAmount

	// When
	openErr := open.UnmarshalJSON([]byte(`"OPEN"`))
	decimalErr := decimal.UnmarshalJSON([]byte(`"50.00"`))

	// Then
	if openErr != nil || decimalErr != nil || open != OpenBudget || decimal != BudgetAmount("50.00") {
		t.Fatalf("budget values = %q/%q, errors = %v/%v", open, decimal, openErr, decimalErr)
	}
}

func TestOfferSelections_Known_requires_every_nonempty_item_known(t *testing.T) {
	// Given
	validAdTypes := AdTypeSelection("BANNER,REWARDED_VIDEO")
	invalidNetwork := NetworkSelection("WIFI,SATELLITE")
	emptyDevices := TargetDeviceSelection("")

	// When / Then
	if !validAdTypes.Known() || invalidNetwork.Known() || emptyDevices.Known() {
		t.Fatalf("Known() = ad:%t network:%t device:%t", validAdTypes.Known(), invalidNetwork.Known(), emptyDevices.Known())
	}
}

func mustDecimalText(t *testing.T, raw string) DecimalText {
	t.Helper()
	value, err := ParseDecimalText(raw)
	if err != nil {
		t.Fatalf("ParseDecimalText(%q) error = %v", raw, err)
	}
	return value
}
