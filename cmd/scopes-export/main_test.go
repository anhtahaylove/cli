// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

//go:build scopeexport

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// TestMarshalBrandScopesDoc_ByteFormatGolden pins the exact byte encoding
// (2-space indent, trailing newline, [] for empty, no HTML escaping) against a
// fixed synthetic doc, independent of the catalog.
func TestMarshalBrandScopesDoc_ByteFormatGolden(t *testing.T) {
	doc := &brandScopesDoc{
		Version: "1.2.3",
		Scopes: map[string]domainScopes{
			"a": {
				I18nName:     i18nText{ZhCn: "甲 & 乙", EnUs: "A & B"},
				I18nDesc:     i18nText{ZhCn: "描述", EnUs: "desc"},
				TenantScopes: []string{},
				UserScopes:   []string{"x:read"},
			},
		},
	}
	got, err := marshalBrandScopesDoc(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n" +
		"  \"version\": \"1.2.3\",\n" +
		"  \"scopes\": {\n" +
		"    \"a\": {\n" +
		"      \"i18n_name\": {\n" +
		"        \"zh_cn\": \"甲 & 乙\",\n" +
		"        \"en_us\": \"A & B\"\n" +
		"      },\n" +
		"      \"i18n_desc\": {\n" +
		"        \"zh_cn\": \"描述\",\n" +
		"        \"en_us\": \"desc\"\n" +
		"      },\n" +
		"      \"tenant_scopes\": [],\n" +
		"      \"user_scopes\": [\n" +
		"        \"x:read\"\n" +
		"      ]\n" +
		"    }\n" +
		"  }\n" +
		"}\n"
	if string(got) != want {
		t.Errorf("byte format mismatch:\n got=%q\nwant=%q", got, want)
	}
}

// TestMarshal_NoHTMLEscape verifies & is not HTML-escaped (SetEscapeHTML(false)).
func TestMarshal_NoHTMLEscape(t *testing.T) {
	doc := &brandScopesDoc{Version: "v", Scopes: map[string]domainScopes{
		"d": {I18nName: i18nText{ZhCn: "a & b"}, TenantScopes: []string{}, UserScopes: []string{}},
	}}
	got, err := marshalBrandScopesDoc(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("a & b")) {
		t.Errorf("& was escaped: %s", got)
	}
}

func TestParseBrandExact(t *testing.T) {
	for _, brand := range []string{"feishu", "lark"} {
		if _, err := parseBrandExact(brand); err != nil {
			t.Errorf("parseBrandExact(%q) unexpected error: %v", brand, err)
		}
	}
	for _, bad := range []string{"", "Feishu", "LARK", "xxx"} {
		if _, err := parseBrandExact(bad); err == nil {
			t.Errorf("parseBrandExact(%q) should error", bad)
		}
	}
}

// TestBuildAndMarshal_MatchesGolden asserts both brands' real output is
// byte-identical to the committed golden. The golden is generated with a fixed
// version so the assertion is stable; when the embedded catalog changes, the
// golden must be regenerated and the downstream baseline synced.
func TestBuildAndMarshal_MatchesGolden(t *testing.T) {
	for _, brand := range []string{"feishu", "lark"} {
		doc, err := buildBrandScopesDoc(brand, "GOLDEN_VERSION")
		if err != nil {
			t.Fatalf("build %s: %v", brand, err)
		}
		got, err := marshalBrandScopesDoc(doc)
		if err != nil {
			t.Fatalf("marshal %s: %v", brand, err)
		}
		want, err := os.ReadFile("testdata/" + brand + ".golden.json")
		if err != nil {
			t.Fatalf("read golden %s: %v", brand, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s output differs from golden (catalog change requires regenerating golden and syncing the downstream baseline)", brand)
		}
	}
}

// TestSchemaInvariants checks the stable JSON shape: every domain has the four
// fields and the scope lists are arrays, never null.
func TestSchemaInvariants(t *testing.T) {
	doc, err := buildBrandScopesDoc("feishu", "v")
	if err != nil {
		t.Fatal(err)
	}
	data, err := marshalBrandScopesDoc(doc)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Version string `json:"version"`
		Scopes  map[string]struct {
			I18nName     map[string]string `json:"i18n_name"`
			I18nDesc     map[string]string `json:"i18n_desc"`
			TenantScopes []string          `json:"tenant_scopes"`
			UserScopes   []string          `json:"user_scopes"`
		} `json:"scopes"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Scopes) == 0 {
		t.Fatal("scopes is empty")
	}
	for d, s := range parsed.Scopes {
		if s.TenantScopes == nil || s.UserScopes == nil {
			t.Errorf("domain %q has a null scope array", d)
		}
	}
}
