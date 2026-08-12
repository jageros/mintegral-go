package mintegral

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCampaignServiceList_sendsAuthenticatedGetJSONBody(t *testing.T) {
	// Given
	credentials := mustCredentials(t, "access", "api")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/open/v1/campaign" {
			t.Errorf("request = %s %s, want GET /api/open/v1/campaign", request.Method, request.URL.Path)
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", request.Header.Get("Content-Type"))
		}
		assertAuthHeaders(t, request, credentials, testTime)
		body := decodeBody(t, request)
		if body["campaign_id"] != "10,11" || body["offer_id"] != float64(12) || body["limit"] != float64(50) {
			t.Errorf("body = %#v, want documented list fields", body)
		}
		writeJSONResponse(t, writer, `{"code":200,"data":{"page":1,"limit":50,"total":1,"list":[{"campaign_id":10,"campaign_name":"test","is_coppa":"YES","promotion_type":"APP","alive_in_store":"YES","preview_url":"https://example.test","product_name":"Test","package_name":"com.test","description":"description","icon":"md5","platform":"ANDROID","category":"GAME","app_size":"12.3","min_version":"1.0","maintain_by":"ADV","status":"RUNNING"}]}}`)
	}))
	defer server.Close()
	client := newServiceClient(t, server.URL, credentials)

	// When
	page, err := client.Campaigns().List(context.Background(), CampaignListRequest{
		CampaignIDs: []CampaignID{10, 11}, OfferID: 12, Page: 1, Limit: 50,
	})
	// Then
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.Total != 1 || len(page.List) != 1 || page.List[0].AppSize != DecimalText("12.3") || page.List[0].Status != CampaignStatusRunning {
		t.Fatalf("List() = %#v, want decoded campaign page", page)
	}
}

func TestCampaignServiceCreate_sendsDocumentedJSON(t *testing.T) {
	// Given
	credentials := mustCredentials(t, "access", "api")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/api/open/v1/campaign" {
			t.Errorf("request = %s %s, want POST /api/open/v1/campaign", request.Method, request.URL.Path)
		}
		assertAuthHeaders(t, request, credentials, testTime)
		body := decodeBody(t, request)
		if body["app_size"] != 12.5 || body["promotion_type"] != "APP" || body["min_version"] != "1.0.0" || body["alive_in_store"] != "YES" || body["platform"] != "ANDROID" {
			t.Errorf("body = %#v, want documented campaign fields", body)
		}
		writeJSONResponse(t, writer, `{"code":200,"data":{"campaign_id":10,"app_size":"12.5"}}`)
	}))
	defer server.Close()
	client := newServiceClient(t, server.URL, credentials)

	// When
	campaign, err := client.Campaigns().Create(context.Background(), testCampaignRequest())
	// Then
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if campaign.CampaignID != 10 || campaign.AppSize != DecimalText("12.5") {
		t.Fatalf("Create() = %#v, want decoded campaign", campaign)
	}
}

func TestCampaignServiceUpdate_sendsDocumentedJSON(t *testing.T) {
	// Given
	credentials := mustCredentials(t, "access", "api")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPut || request.URL.Path != "/api/open/v1/campaign" {
			t.Errorf("request = %s %s, want PUT /api/open/v1/campaign", request.Method, request.URL.Path)
		}
		assertAuthHeaders(t, request, credentials, testTime)
		body := decodeBody(t, request)
		if body["campaign_id"] != float64(10) {
			t.Errorf("body = %#v, want campaign_id", body)
		}
		if _, found := body["preview_url"]; found {
			t.Errorf("body = %#v, must omit create-only preview_url", body)
		}
		writeJSONResponse(t, writer, `{"code":200,"data":{"campaign_id":10}}`)
	}))
	defer server.Close()
	client := newServiceClient(t, server.URL, credentials)

	// When
	campaignName := "Updated"
	campaign, err := client.Campaigns().Update(context.Background(), UpdateCampaignRequest{CampaignID: 10, CampaignName: &campaignName})

	// Then
	if err != nil || campaign.CampaignID != 10 {
		t.Fatalf("Update() = %#v, %v; want campaign ID", campaign, err)
	}
}

func testCampaignRequest() CreateCampaignRequest {
	appSize := DecimalText("12.5")
	return CreateCampaignRequest{
		CampaignName: "Test", IsCOPPA: "YES", PromotionType: "APP", AliveInStore: "YES",
		PreviewURL: "https://example.test", ProductName: "Product", PackageName: "com.example", Description: "description",
		Icon: "md5", Platform: "ANDROID", Category: "GAME", AppSize: &appSize, MinVersion: "1.0.0",
	}
}

func TestCampaignListRequestValidate_rejectsStableBounds(t *testing.T) {
	// Given
	request := CampaignListRequest{CampaignIDs: make([]CampaignID, 51), Limit: 51}

	// When
	err := validateCampaignListRequest(&request)

	// Then
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("validate() error = %v, want ErrInvalidRequest", err)
	}
}

func TestCampaignServiceUpdate_rejectsMissingCampaignID(t *testing.T) {
	// Given
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	request := UpdateCampaignRequest{}

	// When
	_, err = client.Campaigns().Update(context.Background(), request)

	// Then
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Update() error = %v, want ErrInvalidRequest", err)
	}
}

func TestCampaignServiceCreate_rejectsInvalidClosedEnum(t *testing.T) {
	// Given
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	request := testCampaignRequest()
	request.PromotionType = "INVALID"

	// When
	_, err = client.Campaigns().Create(context.Background(), request)

	// Then
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Create() error = %v, want ErrInvalidRequest", err)
	}
}

func TestCampaignServiceUpdate_rejectsInvalidClosedEnum(t *testing.T) {
	// Given
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	promotionType := CampaignPromotionType("INVALID")

	// When
	_, err = client.Campaigns().Update(context.Background(), UpdateCampaignRequest{CampaignID: 10, PromotionType: &promotionType})

	// Then
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Update() error = %v, want ErrInvalidRequest", err)
	}
}

func TestCampaignServiceCreate_rejectsNumericResponseAppSize(t *testing.T) {
	// Given
	credentials := mustCredentials(t, "access", "api")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeJSONResponse(t, writer, `{"code":200,"data":{"campaign_id":10,"app_size":12.5}}`)
	}))
	defer server.Close()
	client := newServiceClient(t, server.URL, credentials)

	// When
	_, err := client.Campaigns().Create(context.Background(), testCampaignRequest())

	// Then
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("Create() error = %v, want ErrUnexpectedResponse", err)
	}
}

func TestCampaignResponseEnums_preserveUnknownValues(t *testing.T) {
	// Given
	wire := campaignWire{
		IsCOPPA: "FUTURE_COPPA", PromotionType: "FUTURE_PROMOTION", AliveInStore: "FUTURE_STORE",
		Platform: "FUTURE_PLATFORM", Status: "FUTURE_STATUS", MaintainBy: "FUTURE_MAINTAINER",
	}

	// When
	campaign, err := parseCampaign(&wire)
	// Then
	if err != nil {
		t.Fatalf("parseCampaign() error = %v", err)
	}
	if campaign.IsCOPPA != "FUTURE_COPPA" || campaign.IsCOPPA.Known() ||
		campaign.PromotionType != "FUTURE_PROMOTION" || campaign.PromotionType.Known() ||
		campaign.AliveInStore != "FUTURE_STORE" || campaign.AliveInStore.Known() ||
		campaign.Platform != "FUTURE_PLATFORM" || campaign.Platform.Known() ||
		campaign.Status != "FUTURE_STATUS" || campaign.Status.Known() ||
		campaign.MaintainBy != "FUTURE_MAINTAINER" || campaign.MaintainBy.Known() {
		t.Fatalf("parseCampaign() = %#v, want preserved unknown enums", campaign)
	}
}

var testTime = time.Unix(1_471_256_697, 0)
