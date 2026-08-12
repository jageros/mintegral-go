package mintegral

import "strings"

// AdType 是官方附录列出的广告展示类型。
type AdType string

// 官方支持的广告展示类型。
const (
	AdTypeBanner              AdType = "BANNER"
	AdTypeDisplayInterstitial AdType = "DISPLAY_INTERSTITIAL"
	AdTypeDisplayNative       AdType = "DISPLAY_NATIVE"
	AdTypeAppWall             AdType = "APPWALL"
	AdTypeMoreOffer           AdType = "MORE_OFFER"
	AdTypeSplash              AdType = "SPLASH_AD"
	AdTypeInterstitialVideo   AdType = "INTERSTITIAL_VIDEO"
	AdTypeNativeVideo         AdType = "NATIVE_VIDEO"
	AdTypeInstreamVideo       AdType = "INSTREAM_VIDEO"
	AdTypeRewardedVideo       AdType = "REWARDED_VIDEO"
)

// Known 报告广告展示类型是否为当前 SDK 已知值。
func (value AdType) Known() bool {
	switch value {
	case AdTypeBanner, AdTypeDisplayInterstitial, AdTypeDisplayNative, AdTypeAppWall, AdTypeMoreOffer, AdTypeSplash,
		AdTypeInterstitialVideo, AdTypeNativeVideo, AdTypeInstreamVideo, AdTypeRewardedVideo:
		return true
	default:
		return false
	}
}

// AdTypeSelection 是逗号分隔的广告展示类型请求值。
type AdTypeSelection string

// Known 报告逗号分隔的所有广告展示类型是否均为当前 SDK 已知值。
func (value AdTypeSelection) Known() bool {
	return knownCommaList(string(value), func(part string) bool { return AdType(part).Known() })
}

func (value AdTypeSelection) valid() bool { return value.Known() }

// Network 是官方附录列出的网络类型。
type Network string

// 官方支持的网络类型。
const (
	Network2G   Network = "2G"
	Network3G   Network = "3G"
	Network4G   Network = "4G"
	Network5G   Network = "5G"
	NetworkWiFi Network = "WIFI"
)

// Known 报告网络类型是否为当前 SDK 已知值。
func (value Network) Known() bool {
	return value == Network2G || value == Network3G || value == Network4G || value == Network5G || value == NetworkWiFi
}

// NetworkSelection 是逗号分隔的网络类型请求值。
type NetworkSelection string

// Known 报告逗号分隔的所有网络类型是否均为当前 SDK 已知值。
func (value NetworkSelection) Known() bool {
	return knownCommaList(string(value), func(part string) bool { return Network(part).Known() })
}

func (value NetworkSelection) valid() bool { return value.Known() }

// TargetDevice 是官方文档列出的目标设备类型。
type TargetDevice string

// 官方支持的目标设备类型。
const (
	TargetDevicePhone  TargetDevice = "PHONE"
	TargetDeviceTablet TargetDevice = "TABLET"
)

// Known 报告目标设备类型是否为当前 SDK 已知值。
func (value TargetDevice) Known() bool {
	return value == TargetDevicePhone || value == TargetDeviceTablet
}

// TargetDeviceSelection 是逗号分隔的目标设备请求值。
type TargetDeviceSelection string

// Known 报告逗号分隔的所有设备类型是否均为当前 SDK 已知值。
func (value TargetDeviceSelection) Known() bool {
	return knownCommaList(string(value), func(part string) bool { return TargetDevice(part).Known() })
}

func (value TargetDeviceSelection) valid() bool { return value.Known() }

// DailyCapType 是每日限额的计量方式。
type DailyCapType string

// 官方支持的每日限额类型。
const (
	DailyCapBudget     DailyCapType = "BUDGET"
	DailyCapConversion DailyCapType = "CONVERSION"
)

// Known 报告每日限额类型是否为当前 SDK 已知值。
func (value DailyCapType) Known() bool { return value == DailyCapBudget || value == DailyCapConversion }

// YesNo 是 Mintegral 请求中的 YES 或 NO 开关。
type YesNo string

// Mintegral 请求使用的布尔开关值。
const (
	Yes YesNo = "YES"
	No  YesNo = "NO"
)

// Known 报告开关是否为 YES 或 NO。
func (value YesNo) Known() bool { return value == Yes || value == No }

// TrackingMethod 是官方附录列出的归因跟踪平台。
type TrackingMethod string

// 官方支持的归因跟踪平台。
const (
	TrackingAdjust      TrackingMethod = "ADJUST"
	TrackingAppsFlyer   TrackingMethod = "APPSFLYER"
	TrackingKochava     TrackingMethod = "KOCHAVA"
	TrackingSingular    TrackingMethod = "SINGULAR"
	TrackingMAT         TrackingMethod = "MAT"
	TrackingTenjin      TrackingMethod = "TENJIN"
	TrackingReyun       TrackingMethod = "REYUN"
	TrackingTalkingData TrackingMethod = "TALKING_DATA"
	TrackingS2S         TrackingMethod = "S2S"
	TrackingUmeng       TrackingMethod = "UMENG"
	TrackingAdmaster    TrackingMethod = "ADMASTER"
	TrackingDataeye     TrackingMethod = "DATAEYE"
	TrackingFox         TrackingMethod = "FOX"
	TrackingPartytrack  TrackingMethod = "PARTYTRACK"
	TrackingAdbrix      TrackingMethod = "ADBRIX"
	TrackingApsalar     TrackingMethod = "APSALAR"
	TrackingAirbridge   TrackingMethod = "AIRBRIDGE"
	TrackingBranch      TrackingMethod = "BRANCH"
)

// Known 报告归因跟踪平台是否为当前 SDK 已知值。
func (value TrackingMethod) Known() bool {
	switch value {
	case TrackingAdjust, TrackingAppsFlyer, TrackingKochava, TrackingSingular, TrackingMAT, TrackingTenjin, TrackingReyun,
		TrackingTalkingData, TrackingS2S, TrackingUmeng, TrackingAdmaster, TrackingDataeye, TrackingFox, TrackingPartytrack,
		TrackingAdbrix, TrackingApsalar, TrackingAirbridge, TrackingBranch:
		return true
	default:
		return false
	}
}

func knownCommaList(value string, known func(string) bool) bool {
	if value == "" {
		return false
	}
	for _, part := range strings.Split(value, ",") {
		if !known(part) {
			return false
		}
	}
	return true
}
