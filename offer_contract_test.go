package mintegral

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestOffers_List_sends_documented_query(t *testing.T) {
	// Given
	client := newOfferContractClient(t, func(request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/open/v1/offers" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		want := "campaign_id=12%2C13&ext_fields=bid_rate_by_mtgid%2Ctarget_app&limit=25&offer_name=launch&page=2&status=RUNNING%2CSTOPPED"
		if got := request.URL.RawQuery; got != want {
			t.Fatalf("query = %q, want %q", got, want)
		}
	})

	// When
	page, err := client.Offers().List(context.Background(), OfferListRequest{
		CampaignIDs:    []CampaignID{12, 13},
		OfferName:      "launch",
		Statuses:       []OfferStatus{OfferStatusRunning, OfferStatusStopped},
		ExtendedFields: []OfferExtendedField{OfferExtendedBidRateByMTGID, OfferExtendedTargetApp},
		Page:           2,
		Limit:          25,
	})

	// Then
	if err != nil || page.Total != 1 || len(page.List) != 1 || page.List[0].OfferID != 99 {
		t.Fatalf("List() = %#v, %v", page, err)
	}
}

func TestOffers_List_rejects_invalid_filter_before_send(t *testing.T) {
	// Given
	var calls atomic.Int64
	client := newOfferContractClient(t, func(*http.Request) { calls.Add(1) })
	tests := []OfferListRequest{
		{CampaignIDs: []CampaignID{0}},
		{OfferIDs: []OfferID{-1}},
		{Statuses: []OfferStatus{"FUTURE_STATUS"}},
		{ExtendedFields: []OfferExtendedField{"unknown"}},
	}

	for _, request := range tests {
		// When
		_, err := client.Offers().List(context.Background(), request)
		// Then
		if !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("List(%#v) error = %v, want ErrInvalidRequest", request, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("transport calls = %d, want 0", calls.Load())
	}
}

func TestOffers_mutations_send_documented_wire_contract(t *testing.T) {
	decimal := mustDecimalText(t, "1.250")
	timezone := mustDecimalText(t, "8")
	zero := mustDecimalText(t, "0")
	dailyBudget := DecimalBudget(mustDecimalText(t, "50"))
	totalBudget := DecimalBudget(mustDecimalText(t, "100"))
	zeroBudget := DecimalBudget(zero)
	emptyLocationBids := []LocationBid{}
	emptyAudienceIDs := []AudienceID{}
	excludedAudienceIDs := []AudienceID{7}
	emptyGeoGoals := []GeoTargetGoal{}
	name := "updated"
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		call   func(*OfferService) error
	}{
		{
			name: "create", method: http.MethodPost, path: "/api/open/v1/offer",
			body: `{"campaign_id":7,"offer_name":"launch","promote_timezone":8,"start_time":1700000000,"target_geo":"US","billing_type":"CPI","target_ad_type":"REWARDED_VIDEO","bid_rate":1.250,"daily_cap_type":"BUDGET","daily_cap":50,"total_budget":100,"os_version_min":"8.0"}`,
			call: func(service *OfferService) error {
				_, err := service.Create(context.Background(), CreateOfferRequest{CampaignID: 7, OfferName: "launch", PromoteTimezone: timezone, StartTime: 1_700_000_000, TargetGeo: "US", BillingType: BillingTypeCPI, TargetAdType: AdTypeSelection(AdTypeRewardedVideo), BidRate: decimal, DailyCapType: DailyCapBudget, DailyCap: &dailyBudget, TotalBudget: &totalBudget, OSVersionMin: "8.0"})
				return err
			},
		},
		{
			name: "update", method: http.MethodPut, path: "/api/open/v1/offer",
			body: `{"offer_id":99,"offer_name":"updated"}`,
			call: func(service *OfferService) error {
				_, err := service.Update(context.Background(), UpdateOfferRequest{OfferID: 99, OfferName: &name})
				return err
			},
		},
		{
			name: "bids preserve empty", method: http.MethodPut, path: "/api/open/v1/offer/bid_rate",
			body: `{"offer_id":99,"bid_rate":1.250,"bid_rate_by_location":[]}`,
			call: func(service *OfferService) error {
				_, err := service.UpdateBids(context.Background(), UpdateOfferBidsRequest{OfferID: 99, BidRate: &decimal, BidRateByLocation: &emptyLocationBids})
				return err
			},
		},
		{
			name: "budget", method: http.MethodPut, path: "/api/open/v1/offer/budget",
			body: `{"offer_id":99,"budget":[{"country_code":"ALL","daily_cap_type":"BUDGET","daily_cap":50,"total_budget":0}]}`,
			call: func(service *OfferService) error {
				return service.UpdateBudget(context.Background(), UpdateOfferBudgetRequest{OfferID: 99, Budget: []OfferBudget{{CountryCode: "ALL", DailyCapType: "BUDGET", DailyCap: dailyBudget, TotalBudget: zeroBudget}}})
			},
		},
		{
			name: "status", method: http.MethodPut, path: "/api/open/v1/offer/status",
			body: `{"offer_id":99,"status":"RUNNING"}`,
			call: func(service *OfferService) error {
				return service.SetStatus(context.Background(), SetOfferStatusRequest{OfferID: 99, Status: OfferStatusRunning})
			},
		},
		{
			name: "traffic", method: http.MethodPut, path: "/api/open/v1/offer/target",
			body: `{"offer_id":99,"option":"ENABLE","mtgid":"mtg1,mtg2"}`,
			call: func(service *OfferService) error {
				return service.UpdateTrafficDelivery(context.Background(), UpdateTrafficDeliveryRequest{OfferID: 99, Option: OfferTargetEnable, MTGIDs: []string{"mtg1", "mtg2"}})
			},
		},
		{
			name: "tracking", method: http.MethodPut, path: "/api/open/v1/tracking",
			body: `{"offer_id":99,"tracking_method":"APPSFLYER","click_url":"https://example.test/click","support_server_click":"YES"}`,
			call: func(service *OfferService) error {
				_, err := service.UpdateTracking(context.Background(), UpdateOfferTrackingRequest{OfferID: 99, TrackingMethod: TrackingAppsFlyer, ClickURL: "https://example.test/click", SupportServerClick: Yes})
				return err
			},
		},
		{
			name: "audiences preserve empty", method: http.MethodPut, path: "/api/open/v1/offer/target-audience",
			body: `{"offer_id":99,"include_ta_id":[],"exclude_ta_id":[7]}`,
			call: func(service *OfferService) error {
				return service.SetAudiences(context.Background(), SetOfferAudiencesRequest{OfferID: 99, IncludeAudienceIDs: &emptyAudienceIDs, ExcludeAudienceIDs: &excludedAudienceIDs})
			},
		},
		{
			name: "target goal", method: http.MethodPut, path: "/api/open/v3/offer/target_goal",
			body: `{"offer_id":99,"target_goal":1.250,"target_goal_by_geo":[]}`,
			call: func(service *OfferService) error {
				return service.UpdateTargetGoal(context.Background(), UpdateOfferTargetGoalRequest{OfferID: 99, TargetGoal: &decimal, TargetGoalByGeo: &emptyGeoGoals})
			},
		},
		{
			name: "legacy creatives", method: http.MethodPut, path: "/api/open/v1/offer/apply_creative",
			body: `{"offer_id":99,"ad_type":"REWARDED_VIDEO","creative":[{"creative_md5":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","creative_name":"video","apply_in_area":"ALL","option":"ENABLE"}]}`,
			call: func(service *OfferService) error {
				_, err := service.ApplyCreatives(context.Background(), ApplyOfferCreativesRequest{OfferID: 99, AdType: AdTypeSelection(AdTypeRewardedVideo), Creatives: []OfferCreative{{CreativeMD5: ContentMD5("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), CreativeName: "video", ApplyInArea: "ALL", Option: OfferTargetEnable}}})
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			client := newOfferContractClient(t, func(request *http.Request) {
				if request.Method != test.method || request.URL.Path != test.path {
					t.Fatalf("request = %s %s, want %s %s", request.Method, request.URL.Path, test.method, test.path)
				}
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				if got := string(body); got != test.body {
					t.Fatalf("body = %s, want %s", got, test.body)
				}
			})

			// When
			err := test.call(client.Offers())
			// Then
			if err != nil {
				t.Fatalf("call error = %v", err)
			}
		})
	}
}

func TestOffers_mutation_rejects_zero_offer_id_before_send(t *testing.T) {
	// Given
	var calls atomic.Int64
	client := newOfferContractClient(t, func(*http.Request) { calls.Add(1) })

	// When
	err := client.Offers().SetStatus(context.Background(), SetOfferStatusRequest{Status: OfferStatusRunning})

	// Then
	if !errors.Is(err, ErrInvalidRequest) || calls.Load() != 0 {
		t.Fatalf("SetStatus() error = %v, calls = %d", err, calls.Load())
	}
}

func TestOffers_Create_resolves_credentials_before_lazy_JSON_encoding(t *testing.T) {
	// Given
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// When
	_, err = client.Offers().Create(context.Background(), CreateOfferRequest{
		CampaignID: 1, OfferName: "launch", StartTime: 1, TargetGeo: "ALL",
		PromoteTimezone: DecimalText("invalid"), BillingType: BillingTypeCPI,
		TargetAdType: AdTypeSelection(AdTypeBanner), BidRate: DecimalText("invalid"),
	})

	// Then
	if !errors.Is(err, ErrCredentialsRequired) {
		t.Fatalf("Create() error = %v, want ErrCredentialsRequired before JSON encoding", err)
	}
}

func TestOffers_Create_rejects_unknown_closed_enum_before_send(t *testing.T) {
	// Given
	var calls atomic.Int64
	client := newOfferContractClient(t, func(*http.Request) { calls.Add(1) })

	// When
	_, err := client.Offers().Create(context.Background(), CreateOfferRequest{
		CampaignID: 1, OfferName: "launch", StartTime: 1, TargetGeo: "ALL",
		BillingType: BillingTypeCPI, TargetAdType: AdTypeSelection("UNKNOWN"),
	})

	// Then
	if !errors.Is(err, ErrInvalidRequest) || calls.Load() != 0 {
		t.Fatalf("Create() error = %v, calls = %d", err, calls.Load())
	}
}

func TestOffers_UpdateTrafficDelivery_allows_all_without_mtgids(t *testing.T) {
	// Given
	client := newOfferContractClient(t, func(request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if got, want := string(body), `{"offer_id":99,"option":"ALLOW_ALL","mtgid":""}`; got != want {
			t.Fatalf("body = %s, want %s", got, want)
		}
	})

	// When
	err := client.Offers().UpdateTrafficDelivery(context.Background(), UpdateTrafficDeliveryRequest{OfferID: 99, Option: OfferTargetAllowAll})
	// Then
	if err != nil {
		t.Fatalf("UpdateTrafficDelivery() error = %v", err)
	}
}

func newOfferContractClient(t *testing.T, inspect func(*http.Request)) *Client {
	t.Helper()
	credentials := mustCredentials(t, "access", "secret")
	client, err := NewClient(
		WithDefaultCredentials(credentials),
		WithClock(fixedClock{}),
		WithAPIBaseURL("https://api.example.test"),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.Header.Get("access-key") != "access" || request.Header.Get("token") == "" {
				t.Fatalf("missing authentication headers")
			}
			inspect(request)
			body := `{"code":200,"msg":"success","data":{}}`
			switch {
			case request.URL.Path == "/api/open/v1/offers":
				body = `{"code":200,"msg":"success","data":{"page":2,"limit":25,"total":1,"list":[{"offer_id":99,"bid_rate":"1.250"}]}}`
			case request.URL.Path == "/api/open/v3/event/bid_goal_supports":
				body = `{"code":200,"msg":"success","data":{"support_events":[{"mtg_event":"Purchase","target_goal_window":["D0"]}]}}`
			case strings.HasSuffix(request.URL.Path, "/apply_creative"):
				body = `{"code":200,"msg":"success","data":["success"]}`
			}
			return jsonResponse(http.StatusOK, body), nil
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}
