package mintegral

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestEvents_BidGoalSupports_sends_documented_query(t *testing.T) {
	// Given
	client := newOfferContractClient(t, func(request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/open/v3/event/bid_goal_supports" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if got, want := request.URL.RawQuery, "bid_goal=Target-ROAS&campaign_id=1234&package_name=com.example.game"; got != want {
			t.Fatalf("query = %q, want %q", got, want)
		}
	})

	// When
	response, err := client.Events().BidGoalSupports(context.Background(), BidGoalSupportsRequest{CampaignID: 1234, PackageName: "com.example.game", BidGoal: BidGoalTargetROAS})

	// Then
	if err != nil || len(response.SupportEvents) != 1 || response.SupportEvents[0].MTGEvent != "Purchase" {
		t.Fatalf("BidGoalSupports() = %#v, %v", response, err)
	}
}

func TestEvents_BidGoalSupports_rejects_missing_selector_before_send(t *testing.T) {
	// Given
	var calls atomic.Int64
	client := newOfferContractClient(t, func(*http.Request) { calls.Add(1) })

	// When
	_, err := client.Events().BidGoalSupports(context.Background(), BidGoalSupportsRequest{BidGoal: BidGoalTargetCPE})

	// Then
	if !errors.Is(err, ErrInvalidRequest) || calls.Load() != 0 {
		t.Fatalf("BidGoalSupports() error = %v, calls = %d", err, calls.Load())
	}
}

func TestEvents_BidGoalSupports_rejects_unknown_bid_goal(t *testing.T) {
	// Given
	client := newOfferContractClient(t, func(*http.Request) {})

	// When
	_, err := client.Events().BidGoalSupports(context.Background(), BidGoalSupportsRequest{CampaignID: 1, BidGoal: BidGoal("unknown")})

	// Then
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("BidGoalSupports() error = %v, want ErrInvalidRequest", err)
	}
}
