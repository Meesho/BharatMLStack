package metrics

import (
	"reflect"
	"testing"
)

func TestNewTag_StoresFields(t *testing.T) {
	tag := NewTag("env", "prod")

	if tag.Name != "env" {
		t.Fatalf("Name = %q, want env", tag.Name)
	}
	if tag.Value != "prod" {
		t.Fatalf("Value = %q, want prod", tag.Value)
	}
}

func TestTagAsString_FormatsAsKeyColonValue(t *testing.T) {
	if got, want := TagAsString("env", "prod"), "env:prod"; got != want {
		t.Fatalf("TagAsString = %q, want %q", got, want)
	}
}

func TestBuildTag_Empty_ReturnsEmptySlice(t *testing.T) {
	got := BuildTag()

	if got == nil {
		t.Fatal("BuildTag() returned nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

func TestBuildTag_Single_ReturnsOneElement(t *testing.T) {
	got := BuildTag(NewTag("env", "prod"))

	want := []string{"env:prod"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTag = %v, want %v", got, want)
	}
}

func TestBuildTag_Many_PreservesOrder(t *testing.T) {
	got := BuildTag(
		NewTag("env", "prod"),
		NewTag("service", "flashring"),
		NewTag("region", "in"),
	)

	want := []string{"env:prod", "service:flashring", "region:in"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildTag = %v, want %v", got, want)
	}
}

func TestUpdateTags_AppendsInOrder(t *testing.T) {
	existing := []string{"env:prod"}

	UpdateTags(&existing, NewTag("service", "flashring"), NewTag("region", "in"))

	want := []string{"env:prod", "service:flashring", "region:in"}
	if !reflect.DeepEqual(existing, want) {
		t.Fatalf("after UpdateTags = %v, want %v", existing, want)
	}
}

func TestUpdateTags_EmptyInput_LeavesUnchanged(t *testing.T) {
	existing := []string{"env:prod"}

	UpdateTags(&existing)

	want := []string{"env:prod"}
	if !reflect.DeepEqual(existing, want) {
		t.Fatalf("existing = %v, want %v", existing, want)
	}
}
