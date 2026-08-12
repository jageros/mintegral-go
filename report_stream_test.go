package mintegral

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestReportStream_decodesEveryDocumentedColumn(t *testing.T) {
	// Given
	header := "Date\tTimestamp\tOffer Id\tOffer Uuid\tOffer Name\tCampaign Id\tCampaign Package\tCreative Id\tCreative Name\tAd Type\tSub Id\tPackage Name\tLocation\tEndcard ID\tEndcard Name\tAd Output Type\tDma Code\tState Code\tCurrency\tImpression\tClick\tConversion\tEcpm\tCpc\tCtr\tCvr\tIvr\tSpend"
	values := "20260812\t1723420800\t11\toffer-uuid\tOffer Name\t22\tcom.campaign\t33\tCreative Name\trewarded\tsub-1\tcom.app\tUS\t44\tEndcard Name\tvideo\t807\tCA\tUSD\t100\t20\t5\t1.2300\t0.4\t0.2\t0.25\t0.05\t9.8700"
	body := io.NopCloser(strings.NewReader(header + "\n" + values + "\n"))
	stream := newReportStream(body, body)
	defer closeReportStream(t, stream)
	date := mustReportDate(t, "2026-08-12")

	// When
	row, err := stream.Next()
	// Then
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	want := ReportRow{Date: date, Timestamp: UnixSeconds(1723420800), OfferID: OfferID(11), OfferUUID: "offer-uuid", OfferName: "Offer Name", CampaignID: CampaignID(22), CampaignPackage: "com.campaign", CreativeID: CreativeID(33), CreativeName: "Creative Name", AdType: "rewarded", SubID: "sub-1", PackageName: "com.app", Location: "US", EndcardID: 44, EndcardName: "Endcard Name", AdOutputType: "video", DmaCode: 807, StateCode: "CA", Currency: "USD", Impressions: 100, Clicks: 20, Conversions: 5, ECPM: DecimalText("1.2300"), CPC: DecimalText("0.4"), CTR: DecimalText("0.2"), CVR: DecimalText("0.25"), IVR: DecimalText("0.05"), Spend: DecimalText("9.8700")}
	if !reflect.DeepEqual(row, want) {
		t.Fatalf("row = %+v, want %+v", row, want)
	}
}

func TestReportExtras_columnsCannotMutateValues(t *testing.T) {
	// Given
	extras := ReportExtras{values: map[string]string{"Future": "value"}}

	// When
	columns := extras.Columns()
	columns[0] = "Changed"

	// Then
	if value, ok := extras.Get("Future"); !ok || value != "value" {
		t.Fatalf("Get(Future) = %q, %v", value, ok)
	}
}

func TestReportStream_rejectsMalformedKnownField(t *testing.T) {
	// Given
	client := reportTestClient(t, func(request *http.Request) (*http.Response, error) {
		if request.URL.Query().Get("type") == "1" {
			return jsonResponse(200, `{"code":200,"data":{"is_complete":true}}`), nil
		}
		return reportResponse(reportHeader + "\n" + reportRow("nan", "0") + "\n"), nil
	})
	stream, err := client.Reports().Open(context.Background(), ReportOpenRequest{Query: reportTestQuery(t)}, WithRequestCredentials(mustCredentials(t, "a", "b")))
	if err != nil {
		t.Fatal(err)
	}
	defer closeReportStream(t, stream)

	// When
	_, err = stream.Next()

	// Then
	if !errors.Is(err, ErrInvalidReport) {
		t.Fatalf("Next() error = %v, want ErrInvalidReport", err)
	}
}
