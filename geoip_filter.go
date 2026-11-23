package main

import (
	"fmt"
	"strings"
)

// Province names accepted by the configuration layer.
var validProvinces = map[string]struct{}{
	"北京市": {}, "天津市": {}, "河北省": {}, "山西省": {},
	"内蒙古自治区": {}, "辽宁省": {}, "吉林省": {}, "黑龙江省": {},
	"上海市": {}, "江苏省": {}, "浙江省": {}, "安徽省": {},
	"福建省": {}, "江西省": {}, "山东省": {}, "河南省": {},
	"湖北省": {}, "湖南省": {}, "广东省": {}, "广西壮族自治区": {},
	"海南省": {}, "重庆市": {}, "四川省": {}, "贵州省": {},
	"云南省": {}, "西藏自治区": {}, "陕西省": {}, "甘肃省": {},
	"青海省": {}, "宁夏回族自治区": {}, "新疆维吾尔自治区": {},
	"台湾省": {}, "香港特别行政区": {}, "澳门特别行政区": {},
}

type ISP string

const (
	ISPUnknown       ISP = "UNKNOWN"
	ISPChinaTelecom  ISP = "CHINATELECOM"
	ISPChinaUnicom   ISP = "CHINAUNICOM"
	ISPChinaMobile   ISP = "CHINAMOBILE"
	ISPChinaBroadnet ISP = "CHINABROADNET"
	ISPCERNET        ISP = "CERNET"
	ISPOther         ISP = "OTHER"
)

// NormalizeProvince converts known IP2Region province abbreviations
// to canonical names, then validates the result.
func NormalizeProvince(value string) (string, bool) {
	value = strings.TrimSpace(value)

	// Unknown or unavailable province values.
	if value == "" || value == "0" || value == "-" ||
		value == "未知" ||
		strings.EqualFold(value, "unknown") ||
		strings.EqualFold(value, "reserved") {
		return "", false
	}

	// Normalize the five autonomous-region abbreviations
	// found in the IP2Region source data.
	switch value {
	case "广西":
		value = "广西壮族自治区"
	case "新疆":
		value = "新疆维吾尔自治区"
	case "内蒙古":
		value = "内蒙古自治区"
	case "宁夏":
		value = "宁夏回族自治区"
	case "西藏":
		value = "西藏自治区"
	}

	if _, ok := validProvinces[value]; !ok {
		return "", false
	}

	return value, true
}

// NormalizeISP maps known IP2Region ISP values to standard categories.
// Unrecognized values are UNKNOWN, not OTHER.

func NormalizeISP(value string) ISP {
	value = strings.TrimSpace(value)

	if value == "" || value == "0" || value == "-" ||
		value == "未知" || strings.EqualFold(value, "unknown") {
		return ISPUnknown
	}

	switch value {
	// 中国电信
	case "电信", "中国电信":
		return ISPChinaTelecom

	// 中国联通
	case "联通", "中国联通":
		return ISPChinaUnicom

	// 中国移动及其铁通体系
	case "移动", "中国移动", "中移铁通", "铁通":
		return ISPChinaMobile

	// 中国广电：只接受明确的中国广电名称
	case "中国广电":
		return ISPChinaBroadnet

	// 中国教育科研网
	case "教育网", "中国教育", "中国教育网",
		"中国教育科研网", "中国教育和科研计算机网":
		return ISPCERNET

	// 已知的其他国内运营商、宽带或网络组织
	case "鹏博士", "数联信息技术", "中国科技网",
		"中华电信", "远传电信", "台湾大哥大电信",
		"台湾固网", "亚太电信", "电讯盈科",
		"香港宽频", "香港有线电视",
		"北京神州长城通信", "东方有线网络",
		"东方有线网络有限公司", "北京歌华有线",
		"歌华有线", "广东广电", "河南广电",
		"内蒙古广电", "河北广电", "湖南有线",
		"贵州广电", "广西广电", "重庆广电",
		"甘肃广电", "宁夏广电", "黑龙江广电",
		"网宿科技", "中企通信", "澳门电讯", "凯擘",
		"中国互联网信息中心", "高能物理研究所",
		"澳洲电信", "澳大利亚电信",
		"雅虎", "CNC集团",
		"深圳市天威视讯",
		"方正宽带", "长城宽带", "上海长城宽带":
		return ISPOther

	// 国内互联网企业及云服务商
	case "阿里", "阿里云",
		"百度", "百度云",
		"腾讯", "腾讯云",
		"华为", "华为云",
		"京东", "网易网络", "字节跳动",
		"爱奇艺", "世纪互联", "优刻得信息",
		"青云科技", "中联数据", "奥飞数据":
		return ISPOther

	// 已知国际互联网、云计算及 CDN 网络
	case "Cloudflare", "Cloudflare, Inc.",
		"Akamai", "Akamai Technologies, Inc.",
		"Akamai International B.V.", "阿卡迈",
		"Amazon", "Amazon.com, Inc.", "亚马逊",
		"Google", "Google LLC", "Google Singapore Pte. Ltd.",
		"谷歌",
		"Microsoft Corporation", "微软",
		"Apple Inc.", "苹果", "苹果科技",
		"Oracle Corporation",
		"IBM云", "Hurricane Electric", "HurricaneElectric",
		"Cogent通信", "GTT通信", "Level3",
		"Zenlayer", "Equinix", "易昆尼克斯",
		"Hurricane Electric LLC",
		"塔塔通信", "美国电话电报",
		"沃达丰", "日本电报电话公司",
		"和记环球电讯",
		"SpaceX":
		return ISPOther

	// 其他名称暂时不猜测
	default:
		return ISPUnknown
	}
}

type RegionMode string

const (
	RegionAll    RegionMode = "ALL"
	RegionLocal  RegionMode = "LOCAL"
	RegionRemote RegionMode = "REMOTE"
)

type ISPMode string

const (
	ISPModeAll       ISPMode = "ALL"
	ISPModeSame      ISPMode = "SAME"
	ISPModeDifferent ISPMode = "DIFFERENT"
	ISPModeSpecific  ISPMode = "SPECIFIC"
)

type IPType string

const (
	IPTypeUnknown     IPType = "UNKNOWN"
	IPTypeResidential IPType = "RESIDENTIAL"
	IPTypeDatacenter  IPType = "DATACENTER"
	IPTypeMobile      IPType = "MOBILE"
	IPTypeEducation   IPType = "EDUCATION"
	IPTypeBusiness    IPType = "BUSINESS"
	IPTypeOther       IPType = "OTHER"
)

type IPTypeMode string

const (
	IPTypeModeAll               IPTypeMode = "ALL"
	IPTypeModeResidential       IPTypeMode = "RESIDENTIAL"
	IPTypeModeDatacenter        IPTypeMode = "DATACENTER"
	IPTypeModeMobile            IPTypeMode = "MOBILE"
	IPTypeModeEducation         IPTypeMode = "EDUCATION"
	IPTypeModeBusiness          IPTypeMode = "BUSINESS"
	IPTypeModeOther             IPTypeMode = "OTHER"
	IPTypeModeExcludeDatacenter IPTypeMode = "EXCLUDE_DATACENTER"
)

// GeoIPFilterConfig defines the three independent filtering dimensions.
type GeoIPFilterConfig struct {
	LocalProvince string
	LocalISP      ISP

	RegionMode RegionMode
	ISPMode    ISPMode
	TargetISP  ISP

	IPTypeMode IPTypeMode
}

// GeoIPFilter evaluates location, ISP and IP type independently.
type GeoIPFilter struct {
	config GeoIPFilterConfig
}

func NewGeoIPFilter(config GeoIPFilterConfig) (*GeoIPFilter, error) {
	if config.RegionMode == "" {
		config.RegionMode = RegionAll
	}
	if config.ISPMode == "" {
		config.ISPMode = ISPModeAll
	}
	if config.IPTypeMode == "" {
		config.IPTypeMode = IPTypeModeAll
	}

	switch config.RegionMode {
	case RegionAll:
	case RegionLocal, RegionRemote:
		province, ok := NormalizeProvince(config.LocalProvince)
		if !ok {
			return nil, fmt.Errorf("invalid local province %q", config.LocalProvince)
		}
		config.LocalProvince = province
	default:
		return nil, fmt.Errorf("invalid region mode %q", config.RegionMode)
	}

	switch config.ISPMode {
	case ISPModeAll:
	case ISPModeSame, ISPModeDifferent:
		if !isComparableISP(config.LocalISP) {
			return nil, fmt.Errorf("local ISP must be a known major ISP for mode %q", config.ISPMode)
		}
	case ISPModeSpecific:
		if !isSelectableISP(config.TargetISP) {
			return nil, fmt.Errorf("invalid target ISP %q", config.TargetISP)
		}
	default:
		return nil, fmt.Errorf("invalid ISP mode %q", config.ISPMode)
	}

	switch config.IPTypeMode {
	case IPTypeModeAll, IPTypeModeResidential, IPTypeModeDatacenter,
		IPTypeModeMobile, IPTypeModeEducation, IPTypeModeBusiness,
		IPTypeModeOther, IPTypeModeExcludeDatacenter:
	default:
		return nil, fmt.Errorf("invalid IP type mode %q", config.IPTypeMode)
	}

	return &GeoIPFilter{config: config}, nil
}

func isComparableISP(isp ISP) bool {
	switch isp {
	case ISPChinaTelecom, ISPChinaUnicom, ISPChinaMobile,
		ISPChinaBroadnet, ISPCERNET:
		return true
	default:
		return false
	}
}

func isSelectableISP(isp ISP) bool {
	return isComparableISP(isp) || isp == ISPOther
}

// Match returns whether an IP passes all configured dimensions.
// The second return value explains the first failed condition.
func (f *GeoIPFilter) Match(info GeoIPInfo, ipType IPType) (bool, string) {
	if f == nil {
		return false, "filter is nil"
	}

	if !f.matchRegion(info) {
		return false, "region does not match"
	}

	if !f.matchISP(info) {
		return false, "ISP does not match"
	}

	if !f.matchIPType(ipType) {
		return false, "IP type does not match"
	}

	return true, "matched"
}

func (f *GeoIPFilter) matchRegion(info GeoIPInfo) bool {
	switch f.config.RegionMode {
	case RegionAll:
		return true
	case RegionLocal, RegionRemote:
		province, ok := NormalizeProvince(info.Province)
		if !ok {
			return false
		}

		local := province == f.config.LocalProvince
		if f.config.RegionMode == RegionLocal {
			return local
		}
		return !local
	default:
		return false
	}
}

func (f *GeoIPFilter) matchISP(info GeoIPInfo) bool {
	remoteISP := NormalizeISP(info.ISP)

	switch f.config.ISPMode {
	case ISPModeAll:
		return true
	case ISPModeSame:
		return remoteISP != ISPUnknown && remoteISP == f.config.LocalISP
	case ISPModeDifferent:
		return remoteISP != ISPUnknown && remoteISP != f.config.LocalISP
	case ISPModeSpecific:
		return remoteISP != ISPUnknown && remoteISP == f.config.TargetISP
	default:
		return false
	}
}

func (f *GeoIPFilter) matchIPType(ipType IPType) bool {
	switch f.config.IPTypeMode {
	case IPTypeModeAll:
		return true
	case IPTypeModeExcludeDatacenter:
		// Strict mode: UNKNOWN is not accepted because it cannot be
		// proven to be non-datacenter.
		return ipType != IPTypeUnknown && ipType != IPTypeDatacenter
	case IPTypeModeResidential:
		return ipType == IPTypeResidential
	case IPTypeModeDatacenter:
		return ipType == IPTypeDatacenter
	case IPTypeModeMobile:
		return ipType == IPTypeMobile
	case IPTypeModeEducation:
		return ipType == IPTypeEducation
	case IPTypeModeBusiness:
		return ipType == IPTypeBusiness
	case IPTypeModeOther:
		return ipType == IPTypeOther
	default:
		return false
	}
}
