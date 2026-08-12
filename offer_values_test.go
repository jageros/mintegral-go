package mintegral

import (
	"errors"
	"reflect"
	"testing"
)

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

func TestBudgetAmount_UnmarshalJSON_clearsPrepopulatedValue_whenJSONNull(t *testing.T) {
	// Given
	budget := OpenBudget

	// When
	err := budget.UnmarshalJSON([]byte(" \n null \t"))

	// Then
	if err != nil || budget != "" {
		t.Fatalf("BudgetAmount.UnmarshalJSON() = %q, %v; want empty budget and nil error", budget, err)
	}
}

func TestBudgetAmount_UnmarshalJSON_rejectsNilReceiver(t *testing.T) {
	// Given
	var budget *BudgetAmount

	// When
	err := budget.UnmarshalJSON([]byte(`"OPEN"`))

	// Then
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("BudgetAmount.UnmarshalJSON() error = %v, want ErrUnexpectedResponse", err)
	}
}

func TestOffer_UnmarshalJSON_clearsPrepopulatedValue_whenJSONNull(t *testing.T) {
	// Given
	offer := Offer{CampaignID: 7, OfferName: "existing"}

	// When
	err := offer.UnmarshalJSON([]byte(" \n null \t"))

	// Then
	if err != nil || !reflect.DeepEqual(offer, Offer{}) {
		t.Fatalf("Offer.UnmarshalJSON() = %#v, %v; want zero offer and nil error", offer, err)
	}
}

func TestOffer_UnmarshalJSON_clearsCampaignID_whenJSONNull(t *testing.T) {
	// Given
	offer := Offer{CampaignID: 7}

	// When
	err := offer.UnmarshalJSON([]byte(`{"campaign_id": null}`))

	// Then
	if err != nil || offer.CampaignID != 0 {
		t.Fatalf("Offer.UnmarshalJSON() campaign ID = %d, %v; want zero campaign ID and nil error", offer.CampaignID, err)
	}
}

func TestOffer_UnmarshalJSON_acceptsNumberAndDecimalStringCampaignID(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{name: "number", data: `{"campaign_id":7}`},
		{name: "decimal string", data: `{"campaign_id":"7"}`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// Given
			var offer Offer

			// When
			err := offer.UnmarshalJSON([]byte(testCase.data))

			// Then
			if err != nil || offer.CampaignID != 7 {
				t.Fatalf("Offer.UnmarshalJSON() = %#v, %v; want campaign ID 7 and nil error", offer, err)
			}
		})
	}
}

func TestOffer_UnmarshalJSON_rejectsInvalidCampaignID(t *testing.T) {
	// Given
	var offer Offer

	// When
	err := offer.UnmarshalJSON([]byte(`{"campaign_id":"7.5"}`))

	// Then
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("Offer.UnmarshalJSON() error = %v, want ErrUnexpectedResponse", err)
	}
}

func TestOffer_UnmarshalJSON_rejectsNilReceiver(t *testing.T) {
	// Given
	var offer *Offer

	// When
	err := offer.UnmarshalJSON([]byte(`{"campaign_id":7}`))

	// Then
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("Offer.UnmarshalJSON() error = %v, want ErrUnexpectedResponse", err)
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
