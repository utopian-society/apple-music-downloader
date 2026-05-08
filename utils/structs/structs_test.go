package structs

import (
	"testing"
)

// Tests for new fields added in this PR:
//   - TagSortOrder bool  (yaml:"tag-sort-order")
//   - TagItunesID  bool  (yaml:"tag-itunes-id")
//   - ALACFix      bool  (yaml:"alac-fix")

func TestConfigSet_NewFieldDefaults(t *testing.T) {
	var cfg ConfigSet
	if cfg.TagSortOrder != false {
		t.Errorf("TagSortOrder default: got %v, want false", cfg.TagSortOrder)
	}
	if cfg.TagItunesID != false {
		t.Errorf("TagItunesID default: got %v, want false", cfg.TagItunesID)
	}
	if cfg.ALACFix != false {
		t.Errorf("ALACFix default: got %v, want false", cfg.ALACFix)
	}
}

func TestConfigSet_NewFieldsCanBeSet(t *testing.T) {
	cfg := ConfigSet{
		TagSortOrder: true,
		TagItunesID:  true,
		ALACFix:      true,
	}
	if !cfg.TagSortOrder {
		t.Error("TagSortOrder should be true")
	}
	if !cfg.TagItunesID {
		t.Error("TagItunesID should be true")
	}
	if !cfg.ALACFix {
		t.Error("ALACFix should be true")
	}
}

func TestConfigSet_NewFieldsIndependentOfEachOther(t *testing.T) {
	// Setting one field must not affect the others.
	cfg := ConfigSet{TagSortOrder: true}
	if cfg.TagItunesID {
		t.Error("TagItunesID should be false when only TagSortOrder is set")
	}
	if cfg.ALACFix {
		t.Error("ALACFix should be false when only TagSortOrder is set")
	}

	cfg2 := ConfigSet{TagItunesID: true}
	if cfg2.TagSortOrder {
		t.Error("TagSortOrder should be false when only TagItunesID is set")
	}
	if cfg2.ALACFix {
		t.Error("ALACFix should be false when only TagItunesID is set")
	}

	cfg3 := ConfigSet{ALACFix: true}
	if cfg3.TagSortOrder {
		t.Error("TagSortOrder should be false when only ALACFix is set")
	}
	if cfg3.TagItunesID {
		t.Error("TagItunesID should be false when only ALACFix is set")
	}
}

func TestConfigSet_NewFieldsToggle(t *testing.T) {
	cfg := ConfigSet{TagSortOrder: true, TagItunesID: true, ALACFix: true}
	cfg.TagSortOrder = false
	cfg.TagItunesID = false
	cfg.ALACFix = false
	if cfg.TagSortOrder || cfg.TagItunesID || cfg.ALACFix {
		t.Error("fields should all be false after toggling")
	}
}

// Verify the fields exist alongside pre-existing fields without conflict.
func TestConfigSet_NewFieldsCoexistWithExistingFields(t *testing.T) {
	cfg := ConfigSet{
		Storefront:   "us",
		ALACFix:      true,
		TagSortOrder: true,
		TagItunesID:  false,
		AlacMax:      10,
	}
	if cfg.Storefront != "us" {
		t.Errorf("Storefront: got %q, want %q", cfg.Storefront, "us")
	}
	if cfg.AlacMax != 10 {
		t.Errorf("AlacMax: got %d, want 10", cfg.AlacMax)
	}
	if !cfg.ALACFix {
		t.Error("ALACFix should be true")
	}
	if !cfg.TagSortOrder {
		t.Error("TagSortOrder should be true")
	}
	if cfg.TagItunesID {
		t.Error("TagItunesID should be false")
	}
}
