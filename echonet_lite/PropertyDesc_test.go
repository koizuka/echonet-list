package echonet_lite

import (
	"bytes"
	"echonet-list/echonet_lite/utils"
	"testing"
)

func TestPropertyDesc_Aliases(t *testing.T) {
	pi := PropertyDesc{
		Aliases: map[string][]byte{"on": {0x30}},
	}
	data, ok := pi.ToEDT("on")
	if !ok {
		t.Error("Expected ToEDT to find alias 'on'")
	}
	if !bytes.Equal(data, []byte{0x30}) {
		t.Errorf("Expected EDT [0x30], got %v", data)
	}
	str := pi.EDTToString([]byte{0x30})
	if str != "on" {
		t.Errorf("Expected EDTToString to return 'on', got '%s'", str)
	}
}

func TestPropertyDesc_Decoder(t *testing.T) {
	// NumberDesc as decoder: value "5" -> byte{5}
	desc := NumberDesc{Min: 0, Max: 10, Offset: 0, Unit: "", EDTLen: 1}
	pi := PropertyDesc{Decoder: desc}
	data, ok := pi.ToEDT("5")
	if !ok {
		t.Error("Expected ToEDT to decode '5'")
	}
	if !bytes.Equal(data, []byte{0x05}) {
		t.Errorf("Expected EDT [0x05], got %v", data)
	}
	str := pi.EDTToString([]byte{0x05})
	if str != "5" {
		t.Errorf("Expected EDTToString to return '5', got '%s'", str)
	}
}

func TestPropertyDesc_AliasPriority(t *testing.T) {
	// Alias "1" overrides decoder from NumberDesc
	desc := NumberDesc{Min: 0, Max: 10, Offset: 0, Unit: "", EDTLen: 1}
	pi := PropertyDesc{
		Aliases: map[string][]byte{"1": {0x09}},
		Decoder: desc,
	}
	data, ok := pi.ToEDT("1")
	if !ok {
		t.Error("Expected ToEDT to find alias '1'")
	}
	if !bytes.Equal(data, []byte{0x09}) {
		t.Errorf("Expected EDT [0x09] from alias, got %v", data)
	}
	str := pi.EDTToString([]byte{0x09})
	if str != "1" {
		t.Errorf("Expected EDTToString to return '1' for alias, got '%s'", str)
	}
}

func TestPropertyDesc_NoMatch(t *testing.T) {
	pi := PropertyDesc{}
	data, ok := pi.ToEDT("foo")
	if ok || data != nil {
		t.Error("Expected ToEDT to return (nil, false) when no alias or decoder")
	}
	str := pi.EDTToString([]byte{0x00})
	if str != "" {
		t.Errorf("Expected EDTToString to return empty string when no decoder, got '%s'", str)
	}
}

func TestNumberDesc_FromInt(t *testing.T) {
	desc := NumberDesc{Min: 1, Max: 3, Offset: 10, Unit: "U", EDTLen: 2}
	// In-range values
	bytes1, ok1 := desc.FromInt(1)
	if !ok1 || !bytes.Equal(bytes1, utils.Uint32ToBytes(11, 2)) {
		t.Errorf("FromInt(1) expected %v, got %v ok=%v", utils.Uint32ToBytes(11, 2), bytes1, ok1)
	}
	bytes2, ok2 := desc.FromInt(3)
	if !ok2 || !bytes.Equal(bytes2, utils.Uint32ToBytes(13, 2)) {
		t.Errorf("FromInt(3) expected %v, got %v ok=%v", utils.Uint32ToBytes(13, 2), bytes2, ok2)
	}
	// Out-of-range
	_, ok0 := desc.FromInt(0)
	if ok0 {
		t.Error("Expected FromInt(0) to be out of range")
	}
}

func TestNumberDesc_ToInt(t *testing.T) {
	desc := NumberDesc{Min: 1, Max: 3, Offset: 10, Unit: "U", EDTLen: 2}
	// Valid EDT
	edt := utils.Uint32ToBytes(12, 2) // raw 12 -> num=12-10=2
	num, unit, ok := desc.ToInt(edt)
	if !ok || num != 2 || unit != "U" {
		t.Errorf("ToInt(%v) expected (2, 'U', true), got (%d, '%s', %v)", edt, num, unit, ok)
	}
	// Invalid length
	_, _, okLen := desc.ToInt([]byte{0x01})
	if okLen {
		t.Error("Expected ToInt with wrong length to return ok=false")
	}
	// Out-of-range raw value
	// raw 5 -> num=5-10=-5 out-of-range
	_, _, okRange := desc.ToInt(utils.Uint32ToBytes(5, 2))
	if okRange {
		t.Error("Expected ToInt with out-of-range raw value to return ok=false")
	}
}

func TestNumberDesc_FromString(t *testing.T) {
	desc := NumberDesc{Min: 0, Max: 100, Offset: 0, Unit: "C", EDTLen: 1}
	// Numeric string
	b1, ok1 := desc.FromString("20")
	if !ok1 || !bytes.Equal(b1, []byte{20}) {
		t.Errorf("FromString('20') expected [20], got %v ok=%v", b1, ok1)
	}
	// With unit
	b2, ok2 := desc.FromString("20C")
	if !ok2 || !bytes.Equal(b2, []byte{20}) {
		t.Errorf("FromString('20C') expected [20], got %v ok=%v", b2, ok2)
	}
	// Invalid string
	_, ok3 := desc.FromString("foo")
	if ok3 {
		t.Error("Expected FromString('foo') to fail")
	}
}

func TestNumberDesc_ToString(t *testing.T) {
	desc := NumberDesc{Min: 0, Max: 50, Offset: 0, Unit: "C", EDTLen: 1}
	// Valid
	s, ok := desc.ToString([]byte{30})
	if !ok || s != "30C" {
		t.Errorf("ToString([30]) expected '30C', got '%s' ok=%v", s, ok)
	}
	// Invalid length
	_, ok2 := desc.ToString([]byte{0, 1})
	if ok2 {
		t.Error("Expected ToString with invalid length to return ok=false")
	}
}

func TestStringDesc_FromString(t *testing.T) {
	desc := StringDesc{MinEDTLen: 3, MaxEDTLen: 5}
	// Shorter than MinEDTLen -> padded
	b1, ok1 := desc.FromString("ab")
	if !ok1 || !bytes.Equal(b1, []byte{'a', 'b', 0}) {
		t.Errorf("FromString('ab') expected padded [a b 0], got %v ok=%v", b1, ok1)
	}
	// Within limits
	b2, ok2 := desc.FromString("abcd")
	if !ok2 || !bytes.Equal(b2, []byte("abcd")) {
		t.Errorf("FromString('abcd') expected 'abcd', got %v ok=%v", b2, ok2)
	}
	// Exceeds MaxEDTLen
	_, ok3 := desc.FromString("abcdef")
	if ok3 {
		t.Error("Expected FromString('abcdef') to fail (exceeds MaxEDTLen)")
	}
	// Empty string
	_, ok4 := desc.FromString("")
	if ok4 {
		t.Error("Expected FromString('') to fail")
	}
}

func TestStringDesc_ToString(t *testing.T) {
	desc := StringDesc{MinEDTLen: 3, MaxEDTLen: 5}
	// NUL-padding trimming when length <= MinEDTLen
	edt1 := []byte{'x', 0, 0}
	s1, _ := desc.ToString(edt1)
	if s1 != "x" {
		t.Errorf("ToString(%v) expected 'x', got '%s'", edt1, s1)
	}
	// No trimming when length > MinEDTLen
	edt2 := []byte{'a', 'b', 0, 'c'}
	s2, _ := desc.ToString(edt2)
	if !bytes.Equal([]byte(s2), edt2) {
		t.Errorf("ToString(%v) expected full string with NUL, got '%s'", edt2, s2)
	}
}

func TestPropertyDesc_GetName(t *testing.T) {
	tests := []struct {
		name     string
		prop     PropertyDesc
		lang     string
		expected string
	}{
		{
			name: "English default when lang is empty",
			prop: PropertyDesc{
				Name:             "Operation status",
				NameTranslations: map[string]string{"ja": "動作状態"},
			},
			lang:     "",
			expected: "Operation status",
		},
		{
			name: "English default when lang is 'en'",
			prop: PropertyDesc{
				Name:             "Operation status",
				NameTranslations: map[string]string{"ja": "動作状態"},
			},
			lang:     "en",
			expected: "Operation status",
		},
		{
			name: "Japanese translation when lang is 'ja'",
			prop: PropertyDesc{
				Name:             "Operation status",
				NameTranslations: map[string]string{"ja": "動作状態"},
			},
			lang:     "ja",
			expected: "動作状態",
		},
		{
			name: "Fallback to English when translation not available",
			prop: PropertyDesc{
				Name:             "Operation status",
				NameTranslations: map[string]string{"ja": "動作状態"},
			},
			lang:     "fr",
			expected: "Operation status",
		},
		{
			name: "No translations map returns English name",
			prop: PropertyDesc{
				Name: "Operation status",
			},
			lang:     "ja",
			expected: "Operation status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.prop.GetName(tt.lang)
			if result != tt.expected {
				t.Errorf("GetName(%q) = %q, want %q", tt.lang, result, tt.expected)
			}
		})
	}
}

func TestPropertyDesc_GetShortName(t *testing.T) {
	tests := []struct {
		name     string
		prop     PropertyDesc
		lang     string
		expected string
	}{
		{
			name: "Short name in English when lang is empty",
			prop: PropertyDesc{
				Name:                  "Measured instantaneous power consumption",
				ShortName:             "Instantaneous power",
				NameTranslations:      map[string]string{"ja": "瞬時電力計測値"},
				ShortNameTranslations: map[string]string{"ja": "瞬時電力"},
			},
			lang:     "",
			expected: "Instantaneous power",
		},
		{
			name: "Short name in Japanese when lang is 'ja'",
			prop: PropertyDesc{
				Name:                  "Measured instantaneous power consumption",
				ShortName:             "Instantaneous power",
				NameTranslations:      map[string]string{"ja": "瞬時電力計測値"},
				ShortNameTranslations: map[string]string{"ja": "瞬時電力"},
			},
			lang:     "ja",
			expected: "瞬時電力",
		},
		{
			name: "Fallback to English short name when translation not available",
			prop: PropertyDesc{
				Name:                  "Measured instantaneous power consumption",
				ShortName:             "Instantaneous power",
				NameTranslations:      map[string]string{"ja": "瞬時電力計測値"},
				ShortNameTranslations: map[string]string{"ja": "瞬時電力"},
			},
			lang:     "fr",
			expected: "Instantaneous power",
		},
		{
			name: "Fallback to full name when no short name defined",
			prop: PropertyDesc{
				Name:             "Operation status",
				NameTranslations: map[string]string{"ja": "動作状態"},
			},
			lang:     "en",
			expected: "Operation status",
		},
		{
			name: "Fallback to English short name when no short name translation",
			prop: PropertyDesc{
				Name:             "Operation status",
				ShortName:        "Status",
				NameTranslations: map[string]string{"ja": "動作状態"},
			},
			lang:     "ja",
			expected: "Status",
		},
		{
			name: "Fallback to English short name then full name",
			prop: PropertyDesc{
				Name:                  "Operation status",
				NameTranslations:      map[string]string{"ja": "動作状態"},
				ShortNameTranslations: map[string]string{"ja": "状態"},
			},
			lang:     "ja",
			expected: "状態",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.prop.GetShortName(tt.lang)
			if result != tt.expected {
				t.Errorf("GetShortName(%q) = %q, want %q", tt.lang, result, tt.expected)
			}
		})
	}
}
