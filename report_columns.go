package mintegral

import "strconv"

func reportSetter(name string) func(*ReportRow, string) error {
	stringField := func(field func(*ReportRow) *string) func(*ReportRow, string) error {
		return func(row *ReportRow, value string) error { *field(row) = value; return nil }
	}
	switch name {
	case "Date":
		return func(row *ReportRow, value string) error {
			if len(value) == 8 {
				value = value[:4] + "-" + value[4:6] + "-" + value[6:]
			}
			parsed, err := ParseDate(value)
			if err == nil {
				row.Date = parsed
			}
			return err
		}
	case "Timestamp":
		return func(row *ReportRow, value string) error {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err == nil {
				row.Timestamp = UnixSeconds(parsed)
			}
			return err
		}
	case "Offer Id":
		return func(row *ReportRow, value string) error {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err == nil {
				row.OfferID = OfferID(parsed)
			}
			return err
		}
	case "Offer Uuid":
		return stringField(func(r *ReportRow) *string { return &r.OfferUUID })
	case "Offer Name":
		return stringField(func(r *ReportRow) *string { return &r.OfferName })
	case "Campaign Id":
		return func(row *ReportRow, value string) error {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err == nil {
				row.CampaignID = CampaignID(parsed)
			}
			return err
		}
	case "Campaign Package":
		return stringField(func(r *ReportRow) *string { return &r.CampaignPackage })
	case "Creative Id":
		return func(row *ReportRow, value string) error {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err == nil {
				row.CreativeID = CreativeID(parsed)
			}
			return err
		}
	case "Creative Name":
		return stringField(func(r *ReportRow) *string { return &r.CreativeName })
	case "Ad Type":
		return stringField(func(r *ReportRow) *string { return &r.AdType })
	case "Sub Id":
		return stringField(func(r *ReportRow) *string { return &r.SubID })
	case "Package Name":
		return stringField(func(r *ReportRow) *string { return &r.PackageName })
	case "Location":
		return stringField(func(r *ReportRow) *string { return &r.Location })
	case "Endcard ID":
		return parseRowInt(func(r *ReportRow) *int64 { return &r.EndcardID })
	case "Endcard Name":
		return stringField(func(r *ReportRow) *string { return &r.EndcardName })
	case "Ad Output Type":
		return stringField(func(r *ReportRow) *string { return &r.AdOutputType })
	case "Dma Code":
		return parseRowInt(func(r *ReportRow) *int64 { return &r.DmaCode })
	case "State Code":
		return stringField(func(r *ReportRow) *string { return &r.StateCode })
	case "Currency":
		return stringField(func(r *ReportRow) *string { return &r.Currency })
	case "Impression":
		return parseRowInt(func(r *ReportRow) *int64 { return &r.Impressions })
	case "Click":
		return parseRowInt(func(r *ReportRow) *int64 { return &r.Clicks })
	case "Conversion":
		return parseRowInt(func(r *ReportRow) *int64 { return &r.Conversions })
	case "Ecpm":
		return parseRowDecimal(func(r *ReportRow) *DecimalText { return &r.ECPM })
	case "Cpc":
		return parseRowDecimal(func(r *ReportRow) *DecimalText { return &r.CPC })
	case "Ctr":
		return parseRowDecimal(func(r *ReportRow) *DecimalText { return &r.CTR })
	case "Cvr":
		return parseRowDecimal(func(r *ReportRow) *DecimalText { return &r.CVR })
	case "Ivr":
		return parseRowDecimal(func(r *ReportRow) *DecimalText { return &r.IVR })
	case "Spend":
		return parseRowDecimal(func(r *ReportRow) *DecimalText { return &r.Spend })
	default:
		return nil
	}
}
