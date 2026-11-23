package main

import "testing"

func TestNormalizeProvince(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		// Canonical province names.
		{"北京市", "北京市", true},
		{"河北省", "河北省", true},
		{"广东省", "广东省", true},
		{"内蒙古自治区", "内蒙古自治区", true},
		{"广西壮族自治区", "广西壮族自治区", true},
		{"新疆维吾尔自治区", "新疆维吾尔自治区", true},
		{"台湾省", "台湾省", true},
		{"香港特别行政区", "香港特别行政区", true},
		{"澳门特别行政区", "澳门特别行政区", true},

		// Autonomous-region abbreviations.
		{"广西", "广西壮族自治区", true},
		{"新疆", "新疆维吾尔自治区", true},
		{"内蒙古", "内蒙古自治区", true},
		{"宁夏", "宁夏回族自治区", true},
		{"西藏", "西藏自治区", true},

		// Whitespace.
		{"  广西  ", "广西壮族自治区", true},
		{"  广东省  ", "广东省", true},

		// Unknown values.
		{"", "", false},
		{"0", "", false},
		{"-", "", false},
		{"未知", "", false},
		{"unknown", "", false},
		{"Reserved", "", false},

		// Non-canonical or non-Chinese province names.
		{"河北", "", false},
		{"广东", "", false},
		{"北京", "", false},
		{"江苏省份", "", false},
		{"California", "", false},
		{"Tokyo", "", false},
		{"武汉", "", false},
		{"广州", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := NormalizeProvince(tt.input)
			if got != tt.want || ok != tt.ok {
				t.Errorf(
					"NormalizeProvince(%q) = (%q, %v), want (%q, %v)",
					tt.input, got, ok, tt.want, tt.ok,
				)
			}
		})
	}
}

func TestNormalizeISP(t *testing.T) {
	tests := []struct {
		raw  string
		want ISP
	}{
		// 中国电信
		{"电信", ISPChinaTelecom},
		{"中国电信", ISPChinaTelecom},

		// 中国联通
		{"联通", ISPChinaUnicom},
		{"中国联通", ISPChinaUnicom},

		// 中国移动及铁通
		{"移动", ISPChinaMobile},
		{"中国移动", ISPChinaMobile},
		{"中移铁通", ISPChinaMobile},
		{"铁通", ISPChinaMobile},

		// 中国广电：必须明确标识
		{"中国广电", ISPChinaBroadnet},
		{"广电", ISPUnknown},
		{"广东广电", ISPOther},
		{"河南广电", ISPOther},

		// 中国教育科研网
		{"教育网", ISPCERNET},
		{"中国教育", ISPCERNET},
		{"中国教育网", ISPCERNET},
		{"中国教育科研网", ISPCERNET},
		{"中国教育和科研计算机网", ISPCERNET},

		// 国内其他网络组织
		{"鹏博士", ISPOther},
		{"中国科技网", ISPOther},
		{"东方有线网络", ISPOther},
		{"方正宽带", ISPOther},
		{"网宿科技", ISPOther},
		{"中企通信", ISPOther},
		{"澳门电讯", ISPOther},
		{"凯擘", ISPOther},

		// 国内互联网企业
		{"阿里", ISPOther},
		{"阿里云", ISPOther},
		{"百度", ISPOther},
		{"百度云", ISPOther},
		{"腾讯", ISPOther},
		{"腾讯云", ISPOther},

		// 国际互联网及云服务商
		{"Google LLC", ISPOther},
		{"Cloudflare", ISPOther},
		{"Akamai", ISPOther},
		{"Amazon", ISPOther},
		{"Microsoft Corporation", ISPOther},
		{"Hurricane Electric LLC", ISPOther},
		{"塔塔通信", ISPOther},
		{"美国电话电报", ISPOther},
		{"沃达丰", ISPOther},
		{"日本电报电话公司", ISPOther},
		{"和记环球电讯", ISPOther},

		// 台湾地区运营商不归入中国大陆运营商
		{"中华电信", ISPOther},
		{"远传电信", ISPOther},
		{"台湾大哥大电信", ISPOther},
		{"中国电信（香港）", ISPUnknown},

		// 未知
		{"0", ISPUnknown},
		{"", ISPUnknown},
		{"-", ISPUnknown},
		{"未知", ISPUnknown},
		{"unknown", ISPUnknown},
		{"未知运营商", ISPUnknown},
		{"尚未收录的网络组织", ISPUnknown},

		// 空白处理
		{"  电信  ", ISPChinaTelecom},

		{"中国互联网信息中心", ISPOther},
		{"高能物理研究所", ISPOther},
		{"澳洲电信", ISPOther},
		{"澳大利亚电信", ISPOther},
		{"雅虎", ISPOther},
		{"CNC集团", ISPOther},
		{"深圳市天威视讯", ISPOther},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := NormalizeISP(tt.raw); got != tt.want {
				t.Errorf("NormalizeISP(%q) = %q, want %q",
					tt.raw, got, tt.want)
			}
		})
	}
}

func TestNewGeoIPFilterValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  GeoIPFilterConfig
		wantErr bool
	}{
		{
			name: "default config",
		},
		{
			name: "local region requires province",
			config: GeoIPFilterConfig{
				RegionMode: RegionLocal,
			},
			wantErr: true,
		},
		{
			name: "same ISP requires local ISP",
			config: GeoIPFilterConfig{
				ISPMode: ISPModeSame,
			},
			wantErr: true,
		},
		{
			name: "specific ISP requires target",
			config: GeoIPFilterConfig{
				ISPMode: ISPModeSpecific,
			},
			wantErr: true,
		},
		{
			name: "unknown IP type mode",
			config: GeoIPFilterConfig{
				IPTypeMode: "INVALID",
			},
			wantErr: true,
		},
		{
			name: "valid local configuration",
			config: GeoIPFilterConfig{
				LocalProvince: "河北省",
				LocalISP:      ISPChinaTelecom,
				RegionMode:    RegionRemote,
				ISPMode:       ISPModeSame,
				IPTypeMode:    IPTypeModeResidential,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGeoIPFilter(tt.config)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewGeoIPFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGeoIPFilterRegion(t *testing.T) {
	tests := []struct {
		name     string
		mode     RegionMode
		province string
		want     bool
	}{
		{"local province", RegionLocal, "河北省", true},
		{"other province", RegionLocal, "广东省", false},
		{"remote province", RegionRemote, "广东省", true},
		{"same province is not remote", RegionRemote, "河北省", false},
		{"unknown province rejected", RegionRemote, "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewGeoIPFilter(GeoIPFilterConfig{
				LocalProvince: "河北省",
				RegionMode:    tt.mode,
			})
			if err != nil {
				t.Fatal(err)
			}

			got, reason := filter.Match(
				GeoIPInfo{Province: tt.province},
				IPTypeUnknown,
			)
			if got != tt.want {
				t.Fatalf("Match() = %v, want %v; reason=%s", got, tt.want, reason)
			}
		})
	}
}

func TestGeoIPFilterISP(t *testing.T) {
	tests := []struct {
		name   string
		mode   ISPMode
		local  ISP
		target ISP
		remote string
		want   bool
	}{
		{"same telecom", ISPModeSame, ISPChinaTelecom, "", "电信", true},
		{"different unicom", ISPModeDifferent, ISPChinaTelecom, "", "联通", true},
		{"different rejects same", ISPModeDifferent, ISPChinaTelecom, "", "电信", false},
		{"same rejects unknown", ISPModeSame, ISPChinaTelecom, "", "0", false},
		{"specific mobile", ISPModeSpecific, "", ISPChinaMobile, "移动", true},
		{"specific other", ISPModeSpecific, "", ISPOther, "腾讯", true},
		{"specific rejects unknown", ISPModeSpecific, "", ISPOther, "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewGeoIPFilter(GeoIPFilterConfig{
				ISPMode:   tt.mode,
				LocalISP:  tt.local,
				TargetISP: tt.target,
			})
			if err != nil {
				t.Fatal(err)
			}

			got, reason := filter.Match(
				GeoIPInfo{ISP: tt.remote},
				IPTypeUnknown,
			)
			if got != tt.want {
				t.Fatalf("Match() = %v, want %v; reason=%s", got, tt.want, reason)
			}
		})
	}
}

func TestGeoIPFilterIPType(t *testing.T) {
	tests := []struct {
		name   string
		mode   IPTypeMode
		ipType IPType
		want   bool
	}{
		{"all accepts residential", IPTypeModeAll, IPTypeResidential, true},
		{"all accepts unknown", IPTypeModeAll, IPTypeUnknown, true},
		{"residential accepts residential", IPTypeModeResidential, IPTypeResidential, true},
		{"residential rejects datacenter", IPTypeModeResidential, IPTypeDatacenter, false},
		{"residential rejects unknown", IPTypeModeResidential, IPTypeUnknown, false},
		{"datacenter accepts datacenter", IPTypeModeDatacenter, IPTypeDatacenter, true},
		{"mobile accepts mobile", IPTypeModeMobile, IPTypeMobile, true},
		{"exclude rejects datacenter", IPTypeModeExcludeDatacenter, IPTypeDatacenter, false},
		{"exclude accepts residential", IPTypeModeExcludeDatacenter, IPTypeResidential, true},
		{"exclude rejects unknown", IPTypeModeExcludeDatacenter, IPTypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewGeoIPFilter(GeoIPFilterConfig{
				IPTypeMode: tt.mode,
			})
			if err != nil {
				t.Fatal(err)
			}

			got, reason := filter.Match(GeoIPInfo{}, tt.ipType)
			if got != tt.want {
				t.Fatalf("Match() = %v, want %v; reason=%s", got, tt.want, reason)
			}
		})
	}
}

func TestGeoIPFilterCombined(t *testing.T) {
	filter, err := NewGeoIPFilter(GeoIPFilterConfig{
		LocalProvince: "河北省",
		LocalISP:      ISPChinaTelecom,
		RegionMode:    RegionRemote,
		ISPMode:       ISPModeSame,
		IPTypeMode:    IPTypeModeResidential,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Same ISP, another province, residential IP.
	got, reason := filter.Match(GeoIPInfo{
		Province: "广东省",
		ISP:      "电信",
	}, IPTypeResidential)
	if !got {
		t.Fatalf("expected matching IP to pass; reason=%s", reason)
	}

	// Same ISP and province, but not residential.
	got, _ = filter.Match(GeoIPInfo{
		Province: "广东省",
		ISP:      "电信",
	}, IPTypeDatacenter)
	if got {
		t.Fatal("datacenter IP should be rejected")
	}

	// Different ISP.
	got, _ = filter.Match(GeoIPInfo{
		Province: "广东省",
		ISP:      "联通",
	}, IPTypeResidential)
	if got {
		t.Fatal("different ISP should be rejected")
	}

	// Unknown IP type.
	got, _ = filter.Match(GeoIPInfo{
		Province: "广东省",
		ISP:      "电信",
	}, IPTypeUnknown)
	if got {
		t.Fatal("unknown IP type should be rejected in residential mode")
	}
}

func TestGeoIPFilterAutonomousRegion(t *testing.T) {
	tests := []struct {
		name     string
		mode     RegionMode
		province string
		want     bool
	}{
		{"local abbreviation", RegionLocal, "广西", true},
		{"local canonical", RegionLocal, "广西壮族自治区", true},
		{"remote abbreviation", RegionRemote, "河北省", true},
		{"remote canonical", RegionRemote, "河北省", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewGeoIPFilter(GeoIPFilterConfig{
				LocalProvince: "广西壮族自治区",
				RegionMode:    tt.mode,
			})
			if err != nil {
				t.Fatal(err)
			}

			got, reason := filter.Match(
				GeoIPInfo{Province: tt.province},
				IPTypeUnknown,
			)
			if got != tt.want {
				t.Fatalf("Match() = %v, want %v; reason=%s",
					got, tt.want, reason)
			}
		})
	}
}
