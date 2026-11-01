package converter

import (
  "strings"
  "testing"
)

func TestConvert(t *testing.T) {
  input := `{
    "version": "2.1.0",
    "runs": [{
      "tool": {
        "driver": {
          "name": "docker scout",
          "version": "1.0",
          "rules": [{
            "id": "CVE-2026-13346",
            "helpUri": "https://nvd.nist.gov/vuln/detail/CVE-2026-13346",
            "properties": {
              "purls": ["pkg:pypi/pip@25.2"],
              "fixed_version": "26.2.0",
              "cvssV3_severity": "MEDIUM"
            }
          }]
        }
      },
      "results": [{
        "ruleId": "CVE-2026-13346",
        "message": {
          "text": "Vulnerable pip version"
        }
      }]
    }]
  }`

  report, err := Convert(strings.NewReader(input), nil)
  if err != nil {
    t.Fatal(err)
  }

  if len(report.Vulnerabilities) != 1 {
    t.Fatalf("expected 1 vulnerability, got %d",
      len(report.Vulnerabilities))
  }

  v := report.Vulnerabilities[0]

  if v.ID != "CVE-2026-13346" {
    t.Errorf("unexpected CVE: %s", v.ID)
  }
  if v.Package != "pip" || v.Version != "25.2" {
    t.Errorf("unexpected package: %+v", v)
  }
  if v.FixVersion != "26.2.0" {
    t.Errorf("unexpected fix version: %s", v.FixVersion)
  }
  if v.Severity != "Medium" {
    t.Errorf("unexpected severity: %s", v.Severity)
  }
  if report.Severity != "Medium" {
    t.Errorf("unexpected report severity: %s", report.Severity)
  }
}

func TestDeduplicate(t *testing.T) {
  input := `{
    "version": "2.1.0",
    "runs": [{
      "tool": {"driver": {
        "rules": [{
          "id": "CVE-2026-13346",
          "properties": {
            "purls": ["pkg:pypi/pip@25.2"]
          }
        }]
      }},
      "results": [
        {"ruleId": "CVE-2026-13346"},
        {"ruleId": "CVE-2026-13346"}
      ]
    }]
  }`

  report, err := Convert(strings.NewReader(input), nil)
  if err != nil {
    t.Fatal(err)
  }

  if len(report.Vulnerabilities) != 1 {
    t.Fatalf("expected 1 unique vulnerability, got %d",
      len(report.Vulnerabilities))
  }
}
