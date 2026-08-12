package mintegral

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestCreativeSetsList_whenCalled_sendsQuery(t *testing.T) {
	client := contractClient(t, func(request *http.Request) *http.Response {
		if request.Method != http.MethodGet || request.URL.Path != "/api/open/v1/creative_sets" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if got := request.URL.Query().Get("offer_id"); got != "42" {
			t.Fatalf("offer_id = %q", got)
		}
		return jsonResponse(200, `{"code":200,"data":{"page":1,"limit":10,"total":0,"list":[]}}`)
	})
	_, err := client.CreativeSets().List(context.Background(), CreativeSetListRequest{OfferID: OfferID(42)})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func TestCreativeSetWrites_whenCalled_sendJSONBodyAndAuth(t *testing.T) {
	tests := []struct {
		name   string
		method string
		call   func(*CreativeSetService) error
	}{
		{name: "create", method: http.MethodPost, call: func(service *CreativeSetService) error {
			_, err := service.Create(context.Background(), CreateCreativeSetRequest{CreativeSetName: "set", AdOutputs: []AdOutput{AdOutputFullScreenImage}, Creatives: []CreativeSetInput{{CreativeName: "ad", CreativeMD5: ContentMD5("900150983cd24fb0d6963f7d28e17f72")}}})
			return err
		}},
		{name: "update", method: http.MethodPut, call: func(service *CreativeSetService) error {
			geos := []string{"US"}
			return service.Update(context.Background(), UpdateCreativeSetRequest{OfferID: 42, CreativeSetName: "set", Geos: &geos})
		}},
		{name: "delete", method: http.MethodDelete, call: func(service *CreativeSetService) error {
			return service.Delete(context.Background(), DeleteCreativeSetRequest{OfferID: 42, CreativeSetName: "set"})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := contractClient(t, func(request *http.Request) *http.Response {
				if request.Method != test.method || request.URL.Path != "/api/open/v1/creative_set" || request.Header.Get("access-key") == "" {
					t.Fatalf("request = %s %s auth=%q", request.Method, request.URL.Path, request.Header.Get("access-key"))
				}
				var body struct {
					CreativeSetName string `json:"creative_set_name"`
				}
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body.CreativeSetName != "set" {
					t.Fatalf("body = %#v, error = %v", body, err)
				}
				return jsonResponse(200, `{"code":200,"data":{}}`)
			})
			if err := test.call(client.CreativeSets()); err != nil {
				t.Fatalf("write error = %v", err)
			}
		})
	}
}

func TestCreativeSetUpdate_whenFieldsNil_omitsReplaceFields(t *testing.T) {
	client := contractClient(t, func(request *http.Request) *http.Response {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"geos", "ad_outputs", "creatives"} {
			if _, exists := body[field]; exists {
				t.Fatalf("field %q unexpectedly present", field)
			}
		}
		return jsonResponse(200, `{"code":200,"data":{}}`)
	})
	if err := client.CreativeSets().Update(context.Background(), UpdateCreativeSetRequest{OfferID: 42, CreativeSetName: "set"}); err != nil {
		t.Fatal(err)
	}
}

func TestCreativeSetUpdate_whenFieldsEmpty_sendsEmptyArrays(t *testing.T) {
	emptyGeos := []string{}
	emptyOutputs := []AdOutput{}
	emptyCreatives := []CreativeSetEdit{}
	client := contractClient(t, func(request *http.Request) *http.Response {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"geos", "ad_outputs", "creatives"} {
			if string(body[field]) != "[]" {
				t.Fatalf("field %q = %s, want []", field, body[field])
			}
		}
		return jsonResponse(200, `{"code":200,"data":{}}`)
	})
	request := UpdateCreativeSetRequest{OfferID: 42, CreativeSetName: "set", Geos: &emptyGeos, AdOutputs: &emptyOutputs, Creatives: &emptyCreatives}
	if err := client.CreativeSets().Update(context.Background(), request); err != nil {
		t.Fatal(err)
	}
}

func TestResponseEnumsKnown_whenUnknown_preservesUnknownValues(t *testing.T) {
	if CreativeAuditStatus(99).Known() || CreativeType("FUTURE").Known() || CombinationMethod(99).Known() || AdOutput(999).Known() || AssetPlatform("FUTURE").Known() {
		t.Fatal("unknown enum reported as known")
	}
	if !CreativeAuditApproved.Known() || !CreativeTypeVideo.Known() || !CombinationCustomized.Known() || !AdOutputPlayable.Known() || !AssetPlatformIOS.Known() {
		t.Fatal("documented enum reported as unknown")
	}
}

func TestAssetList_whenCreativeTypeUnknown_rejectsBeforeSend(t *testing.T) {
	client := contractClient(t, func(*http.Request) *http.Response {
		t.Fatal("transport must not be called")
		return nil
	})
	_, err := client.Assets().List(context.Background(), AssetListRequest{CreativeType: CreativeType("FUTURE")})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("List() error = %v, want ErrInvalidRequest", err)
	}
}

func TestCreativeSetCreate_whenRequestEnumInvalid_rejectsBeforeSend(t *testing.T) {
	tests := []CreateCreativeSetRequest{
		{CreativeSetName: "set", CombinationMethod: 3, AdOutputs: []AdOutput{AdOutputFullScreenImage}, Creatives: []CreativeSetInput{{CreativeName: "ad", CreativeMD5: ContentMD5("900150983cd24fb0d6963f7d28e17f72")}}},
		{CreativeSetName: "set", CombinationMethod: 1, AdOutputs: []AdOutput{0}, Creatives: []CreativeSetInput{{CreativeName: "ad", CreativeMD5: ContentMD5("900150983cd24fb0d6963f7d28e17f72")}}},
	}
	for _, request := range tests {
		client := contractClient(t, func(*http.Request) *http.Response {
			t.Fatal("transport must not be called")
			return nil
		})
		_, err := client.CreativeSets().Create(context.Background(), request)
		if !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("Create() error = %v, want ErrInvalidRequest", err)
		}
	}
}

func TestCreativeSetUpdate_whenEditOptionUnknown_rejectsBeforeSend(t *testing.T) {
	edits := []CreativeSetEdit{{CreativeName: "ad", CreativeMD5: ContentMD5("900150983cd24fb0d6963f7d28e17f72"), Option: CreativeSetEditOption("FUTURE")}}
	client := contractClient(t, func(*http.Request) *http.Response {
		t.Fatal("transport must not be called")
		return nil
	})
	err := client.CreativeSets().Update(context.Background(), UpdateCreativeSetRequest{OfferID: 42, CreativeSetName: "set", Creatives: &edits})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Update() error = %v, want ErrInvalidRequest", err)
	}
}

func TestCreativeSetList_whenCombinationUnknown_rejectsBeforeSend(t *testing.T) {
	client := contractClient(t, func(*http.Request) *http.Response {
		t.Fatal("transport must not be called")
		return nil
	})
	_, err := client.CreativeSets().List(context.Background(), CreativeSetListRequest{CombinationMethod: CombinationMethod(99)})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("List() error = %v, want ErrInvalidRequest", err)
	}
}

func TestCreativeSetUpdate_whenAdOutputInvalid_rejectsBeforeSend(t *testing.T) {
	outputs := []AdOutput{0}
	client := contractClient(t, func(*http.Request) *http.Response {
		t.Fatal("transport must not be called")
		return nil
	})
	err := client.CreativeSets().Update(context.Background(), UpdateCreativeSetRequest{OfferID: 42, CreativeSetName: "set", AdOutputs: &outputs})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Update() error = %v, want ErrInvalidRequest", err)
	}
}

func TestCreativeAdsList_whenCalled_sendsGETJSONBody(t *testing.T) {
	client := contractClient(t, func(request *http.Request) *http.Response {
		if request.Method != http.MethodGet || request.URL.Path != "/api/open/v1/creative-ad/list" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		var body CreativeAdListRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.AdIDs) != 2 || request.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("body/header = %#v %q", body, request.Header.Get("Content-Type"))
		}
		return jsonResponse(200, `{"code":200,"message":"success","data":{"page":1,"limit":20,"total":0,"list":[]}}`)
	})
	_, err := client.CreativeAds().List(context.Background(), CreativeAdListRequest{AdIDs: []AdID{1, 2}})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
}

func contractClient(t *testing.T, handle func(*http.Request) *http.Response) *Client {
	t.Helper()
	credentials := mustCredentials(t, "access", "api")
	client, err := NewClient(WithDefaultCredentials(credentials), WithAPIBaseURL("https://api.example.test"), WithStorageBaseURL("https://storage.example.test"), WithHTTPClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) { return handle(request), nil })}))
	if err != nil {
		t.Fatal(err)
	}
	return client
}
