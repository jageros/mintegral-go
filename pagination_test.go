package mintegral

import "testing"

func TestPageInfo_Next_advancesWhenTotalHasMoreRows(t *testing.T) {
	// Given
	page := PageInfo{Number: 1, Limit: 100, Total: 101, Returned: 100}

	// When
	next, ok := page.Next()

	// Then
	if !ok {
		t.Fatal("PageInfo.Next() ok = false, want true")
	}
	if want := (PageRequest{Number: 2, Limit: 100}); next != want {
		t.Fatalf("PageInfo.Next() = %#v, want %#v", next, want)
	}
}

func TestPageInfo_Next_stopsAtLastOrEmptyPage(t *testing.T) {
	tests := []struct {
		name string
		page PageInfo
	}{
		{name: "last page", page: PageInfo{Number: 2, Limit: 100, Total: 101, Returned: 1}},
		{name: "empty page", page: PageInfo{Number: 1, Limit: 100, Total: 101, Returned: 0}},
		{name: "invalid metadata", page: PageInfo{Number: 0, Limit: 100, Total: 101, Returned: 100}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// When
			_, ok := test.page.Next()

			// Then
			if ok {
				t.Fatalf("PageInfo.Next() ok = true, want false for %s", test.name)
			}
		})
	}
}
