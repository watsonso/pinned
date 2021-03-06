package pinned

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionManagerAdd(t *testing.T) {
	vm := &VersionManager{}

	// Should fail if invalid date supplied.
	err := vm.Add(&Version{
		Date: "baddate",
	})
	if err == nil {
		t.Fatal("Expected error when adding version with invalid date")
	}

	// Versions should be sorted in descending order.
	vm.Add(&Version{
		Date: "2016-01-02",
	})
	vm.Add(&Version{
		Date: "2017-01-02",
	})
	versions := vm.Versions()
	if versions[0] != "2017-01-02" || versions[1] != "2016-01-02" {
		t.Fatal("Versions not sorted properly")
	}
}

func TestVersionManagerLatest(t *testing.T) {
	vm := &VersionManager{}

	// If no versions, return nil Version.
	v := vm.Latest()
	if v != nil {
		log.Fatal("Expected nil Version")
	}

	vm.Add(&Version{
		Date: "2017-01-02",
	})
	vm.Add(&Version{
		Date: "2018-01-02",
	})

	v = vm.Latest()
	if v.Date != "2018-01-02" {
		t.Fatalf("Expected 2018-01-02, instead got %s", v.Date)
	}
}

func TestVersionManagerParse(t *testing.T) {
	vm := &VersionManager{}

	getRoute := func(v string) string {
		route := "/users"
		if v != "" {
			route += fmt.Sprintf("?v=%s", v)
		}
		return route
	}

	// Should fail if no version supplied.
	oldV := &Version{
		Date: "2017-01-02",
	}
	vm.Add(oldV)

	req := httptest.NewRequest(http.MethodGet, getRoute(""), nil)
	_, err := vm.Parse(req)
	if err != ErrNoVersionSupplied {
		t.Fatalf("Expected ErrNoVersionSupplied, instead got %s", err)
	}

	// Should fail if invalid version supplied in query params.
	req = httptest.NewRequest(http.MethodGet, getRoute("2000-01-02"), nil)
	_, err = vm.Parse(req)
	if err != ErrInvalidVersion {
		t.Fatalf("Expected ErrInvalidVersion, instead got %s", err)
	}

	// Should fail if invalid version supplied in req header.
	req = httptest.NewRequest(http.MethodGet, getRoute(""), nil)
	req.Header.Set("Version", "2017-01-02")
	_, err = vm.Parse(req)
	if err != nil {
		t.Fatalf("Expected no error, instead got %s", err)
	}

	// Should succeed if valid version supplied in query params.
	req = httptest.NewRequest(http.MethodGet, getRoute("2017-01-02"), nil)
	_, err = vm.Parse(req)
	if err != nil {
		t.Fatalf("Expected no error, instead got %s", err)
	}

	// Should succeed if valid version supplied in req header.
	req = httptest.NewRequest(http.MethodGet, getRoute(""), nil)
	req.Header.Set("Version", "2017-01-02")
	_, err = vm.Parse(req)
	if err != nil {
		t.Fatalf("Expected no error, instead got %s", err)
	}

	// Should select more recent version if supplied in query params and header.
	newV := &Version{
		Date: "2018-01-02",
	}
	vm.Add(newV)

	req = httptest.NewRequest(http.MethodGet, getRoute("2017-01-02"), nil)
	req.Header.Set("Version", "2018-01-02")
	v, err := vm.Parse(req)
	if err != nil {
		t.Fatalf("Expected no error, instead got %s", err)
	}
	if v.Date != "2018-01-02" {
		t.Fatalf("Expected version 2018-01-02, instead got %s", v.Date)
	}

	// Should fail if version is deprecated.
	oldV.Deprecated = true
	req = httptest.NewRequest(http.MethodGet, getRoute("2017-01-02"), nil)
	_, err = vm.Parse(req)
	if err != ErrVersionDeprecated {
		t.Fatalf("Expected ErrVersionDeprecated, instead got %s", err)
	}
}

type TestObject struct {
	B string
}

func (to *TestObject) Data() map[string]interface{} {
	return map[string]interface{}{
		"B": to.B,
	}
}

func TestVersionManagerApply(t *testing.T) {
	vm := &VersionManager{}

	action := func(m map[string]interface{}) map[string]interface{} {
		m["A"] = m["B"]
		delete(m, "B")
		return m
	}

	version := &Version{
		Date: "2017-01-02",
	}
	vm.Add(version)

	vm.Add(&Version{
		Date: "2018-01-02",
		Changes: []*Change{
			{
				Description: "Foobar.",
				Actions: map[string]Action{
					"TestObject": action,
				},
			},
		},
	})

	to := &TestObject{"Foo"}

	res, err := vm.Apply(version, to)
	if err != nil {
		t.Fatal(err)
	}

	if res["A"].(string) != "Foo" {
		t.Fatalf("Expected map[A] = Foo, instead got %s", res["A"].(string))
	}
}
