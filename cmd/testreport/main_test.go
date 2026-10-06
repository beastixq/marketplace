package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadReport(t *testing.T) {
	t.Run("counts leaf cases without parents", func(t *testing.T) {
		input := strings.NewReader(`
{"Action":"output","Package":"p","Output":"-test.shuffle 42\n"}
{"Action":"pass","Package":"p","Test":"TestService/Method/success"}
{"Action":"skip","Package":"p","Test":"TestService/Method/skipped"}
{"Action":"pass","Package":"p","Test":"TestService/Method"}
{"Action":"pass","Package":"p","Test":"TestService"}
{"Action":"pass","Package":"p","Test":"TestStandalone"}
{"Action":"pass","Package":"p"}
`)

		got, err := readReport(input)

		if err != nil {
			t.Fatal(err)
		}
		if got.Passed != 2 || got.Skipped != 1 || got.Failed != 0 || got.Incomplete != 0 {
			t.Fatalf("unexpected leaf counts: %+v", got)
		}
		if len(got.Packages) != 1 || len(got.Packages[0].Tests) != 3 || got.Packages[0].Seed != "42" {
			t.Fatalf("unexpected package: %+v", got.Packages)
		}
	})

	t.Run("retains build failure with no test cases", func(t *testing.T) {
		input := strings.NewReader(`
{"Action":"build-output","ImportPath":"broken","Output":"undefined: symbol\n"}
{"Action":"build-fail","ImportPath":"broken"}
`)

		got, err := readReport(input)

		if err != nil {
			t.Fatal(err)
		}
		if got.FailedPackages != 1 || got.Passed != 0 || len(got.Packages[0].Tests) != 0 {
			t.Fatalf("build failure hidden in report: %+v", got)
		}
		if got.Packages[0].Output != "undefined: symbol\n" {
			t.Fatalf("missing build output: %q", got.Packages[0].Output)
		}
	})

	t.Run("retains failure in parent cleanup", func(t *testing.T) {
		input := strings.NewReader(`
{"Action":"pass","Package":"p","Test":"TestService/success"}
{"Action":"output","Package":"p","Test":"TestService","Output":"cleanup failed\n"}
{"Action":"fail","Package":"p","Test":"TestService"}
{"Action":"fail","Package":"p"}
`)

		got, err := readReport(input)

		if err != nil {
			t.Fatal(err)
		}
		if got.Passed != 1 || got.FailedPackages != 1 || !strings.Contains(got.Packages[0].Output, "cleanup failed") {
			t.Fatalf("parent failure hidden: %+v", got)
		}
	})

	for _, tc := range []struct {
		name  string
		input string
	}{
		{name: "rejects empty input"},
		{name: "rejects malformed input", input: "{not json}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readReport(strings.NewReader(tc.input))

			if err == nil {
				t.Fatalf("expected error for %q", tc.input)
			}
		})
	}
}

func TestRenderReport(t *testing.T) {
	results := report{Failed: 1, FailedPackages: 1, Packages: []*packageResult{{
		Name: "p", Status: "fail", Tests: []*testResult{{
			Name: "TestService/<script>alert(1)</script>", Status: "fail",
			Output: "unexpected <img src=x onerror=alert(1)>",
		}},
	}}}
	var html bytes.Buffer

	err := renderReport(&html, results)

	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"<script>alert(1)</script>", "<img src=x onerror=alert(1)>"} {
		if strings.Contains(html.String(), raw) {
			t.Fatalf("unescaped test content: %s", raw)
		}
	}
	if !strings.Contains(html.String(), "&lt;img src=x onerror=alert(1)&gt;") || !strings.Contains(html.String(), "FAIL: 1") {
		t.Fatal("escaped failure details or failure count missing")
	}
}
