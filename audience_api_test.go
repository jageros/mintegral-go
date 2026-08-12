package mintegral

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestAudienceList_encodesQueryAndDecodesDocumentedFields(t *testing.T) {
	// Given
	client := audienceTestClient(t, func(request *http.Request) *http.Response {
		if got := request.URL.RequestURI(); got != "/api/open/v1/audience?limit=500&page=2&platform=2&ta_ids=12%2C34&ta_name=vip" {
			t.Errorf("request URI = %q", got)
		}
		return jsonResponse(http.StatusOK, `{"code":200,"message":"success","data":{"page":2,"limit":500,"current_total":1,"total":3,"list":[{"ta_id":12,"platform":2,"ta_name":"vip-ios","ta_type":7,"device_type":"2,10,11,12,13","area_type":1,"ctime":1704871629,"utime":1704871631}]}}`)
	})

	// When
	result, err := client.Audiences().List(context.Background(), AudienceListRequest{TAIDs: []AudienceID{12, 34}, TAName: "vip", Platform: 2, Limit: 500, Page: 2}, WithRequestCredentials(mustCredentials(t, "a", "b")))
	// Then
	if err != nil {
		t.Fatalf("ListAudience() error = %v", err)
	}
	if result.Total != 3 || len(result.List) != 1 || result.List[0].DeviceType != "2,10,11,12,13" {
		t.Fatalf("ListAudience() = %#v", result)
	}
}

func TestAudienceMutations_sendDocumentedJSONBodies(t *testing.T) {
	// Given
	requests := make(chan *http.Request, 3)
	bodies := make(chan string, 3)
	client := audienceTestClient(t, func(request *http.Request) *http.Response {
		body, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			t.Fatal(readErr)
		}
		requests <- request
		bodies <- string(body)
		if request.Method == http.MethodDelete {
			return jsonResponse(http.StatusOK, `{"code":200,"msg":"success","data":null}`)
		}
		return jsonResponse(http.StatusOK, `{"code":200,"message":"success","data":{"ta_id":1148}}`)
	})
	credential := WithRequestCredentials(mustCredentials(t, "a", "b"))
	paths := []AudienceDataPath{{DeviceType: AudienceDeviceType(12), DataPath: "s3://bucket/idfv.csv"}}

	// When
	service := client.Audiences()
	created, createErr := service.Create(context.Background(), CreateAudienceRequest{TAName: "vip", AreaType: 1, Platform: 2, DataPath: paths}, credential)
	updated, updateErr := service.Update(context.Background(), UpdateAudienceRequest{TAID: 1148, DataPath: paths}, credential)
	deleteErr := service.Delete(context.Background(), DeleteAudienceRequest{TAIDs: []AudienceID{1148}}, credential)

	// Then
	if createErr != nil || updateErr != nil || deleteErr != nil || created.TAID != 1148 || updated.TAID != 1148 {
		t.Fatalf("mutation results create=%#v/%v update=%#v/%v delete=%v", created, createErr, updated, updateErr, deleteErr)
	}
	wants := []struct{ method, body string }{
		{http.MethodPost, `{"ta_name":"vip","area_type":1,"platform":2,"data_path":[{"device_type":12,"data_path":"s3://bucket/idfv.csv"}]}`},
		{http.MethodPut, `{"ta_id":1148,"data_path":[{"device_type":12,"data_path":"s3://bucket/idfv.csv"}]}`},
		{http.MethodDelete, `{"ta_id":[1148]}`},
	}
	for _, want := range wants {
		request := <-requests
		body := <-bodies
		if request.Method != want.method || body != want.body {
			t.Errorf("request = %s %s, want %s %s", request.Method, body, want.method, want.body)
		}
	}
}

func audienceTestClient(t *testing.T, handle func(*http.Request) *http.Response) *Client {
	t.Helper()
	client, err := NewClient(
		WithAPIBaseURL("https://api.example.test"),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.Header.Get("access-key") == "" {
				t.Error("authenticated request lacks access-key")
			}
			return handle(request), nil
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}
