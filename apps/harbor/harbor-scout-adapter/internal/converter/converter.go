package converter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"
)

const ReportMimeType = "application/vnd.scanner.adapter.vuln.report.harbor+json; version=1.0"

type SARIF struct {
	Version string `json:"version"`
	Runs    []struct {
		Tool struct {
			Driver struct {
				Name    string `json:"name"`
				Version string `json:"version"`
				Rules   []Rule `json:"rules"`
			} `json:"driver"`
		} `json:"tool"`
		Results []Result `json:"results"`
	} `json:"runs"`
}

type Rule struct {
	ID               string     `json:"id"`
	HelpURI          string     `json:"helpUri"`
	ShortDescription Message    `json:"shortDescription"`
	FullDescription  Message    `json:"fullDescription"`
	Properties       Properties `json:"properties"`
}

type Message struct {
	Text string `json:"text"`
}

type Properties struct {
	PURLs        []string `json:"purls"`
	FixedVersion string   `json:"fixed_version"`
	CVSSSeverity string   `json:"cvssV3_severity"`
	Severity     string   `json:"severity"`
}

type Result struct {
	RuleID     string     `json:"ruleId"`
	RuleIndex  *int       `json:"ruleIndex"`
	Message    Message    `json:"message"`
	Properties Properties `json:"properties"`
}

type Scanner struct {
	Name    string `json:"name"`
	Vendor  string `json:"vendor"`
	Version string `json:"version"`
}

type Artifact struct {
	Repository string `json:"repository,omitempty"`
	Digest     string `json:"digest,omitempty"`
	Tag        string `json:"tag,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
}

type Vulnerability struct {
	ID          string   `json:"id"`
	Package     string   `json:"package"`
	Version     string   `json:"version"`
	FixVersion  string   `json:"fix_version,omitempty"`
	Severity    string   `json:"severity"`
	Description string   `json:"description,omitempty"`
	Links       []string `json:"links"`
}

type Report struct {
	GeneratedAt     time.Time       `json:"generated_at"`
	Artifact        *Artifact       `json:"artifact,omitempty"`
	Scanner         Scanner         `json:"scanner"`
	Severity        string          `json:"severity"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
}

func Convert(r io.Reader, artifact *Artifact) (*Report, error) {
	var input SARIF
	if err := json.NewDecoder(r).Decode(&input); err != nil {
		return nil, fmt.Errorf("decode SARIF: %w", err)
	}

	if input.Version != "2.1.0" {
		return nil, fmt.Errorf("unsupported SARIF version: %q", input.Version)
	}

	report := &Report{
		GeneratedAt: time.Now().UTC(),
		Artifact:    artifact,
		Scanner: Scanner{
			Name:    "Docker Scout",
			Vendor:  "Docker",
			Version: "unknown",
		},
		Severity:        "Unknown",
		Vulnerabilities: []Vulnerability{},
	}

	seen := make(map[string]bool)

	for _, run := range input.Runs {
		if run.Tool.Driver.Version != "" {
			report.Scanner.Version = run.Tool.Driver.Version
		}

		rules := make(map[string]Rule)
		for _, rule := range run.Tool.Driver.Rules {
			rules[rule.ID] = rule
		}

		for _, result := range run.Results {
			rule, found := rules[result.RuleID]

			if !found && result.RuleIndex != nil {
				i := *result.RuleIndex
				if i >= 0 && i < len(run.Tool.Driver.Rules) {
					rule = run.Tool.Driver.Rules[i]
					found = true
				}
			}

			if !found {
				return nil, fmt.Errorf(
					"result references unknown rule %q",
					result.RuleID,
				)
			}

			id := rule.ID
			if id == "" {
				id = result.RuleID
			}

			if id == "" {
				return nil, fmt.Errorf("result without vulnerability ID")
			}

			purls := result.Properties.PURLs
			if len(purls) == 0 {
				purls = rule.Properties.PURLs
			}

			if len(purls) == 0 {
				return nil, fmt.Errorf(
					"vulnerability %s has no package PURL",
					id,
				)
			}

			fix := result.Properties.FixedVersion
			if fix == "" {
				fix = rule.Properties.FixedVersion
			}

			severity := result.Properties.CVSSSeverity
			if severity == "" {
				severity = result.Properties.Severity
			}
			if severity == "" {
				severity = rule.Properties.CVSSSeverity
			}
			if severity == "" {
				severity = rule.Properties.Severity
			}
			severity = normalizeSeverity(severity)

			description := result.Message.Text
			if description == "" {
				description = rule.FullDescription.Text
			}
			if description == "" {
				description = rule.ShortDescription.Text
			}

			links := []string{}
			if rule.HelpURI != "" {
				link := strings.TrimSpace(rule.HelpURI)

				if u, err := url.ParseRequestURI(link); err == nil &&
					(u.Scheme == "https" || u.Scheme == "http") &&
					u.Host != "" {
					links = append(links, link)
				}
			}

			for _, purl := range purls {
				name, version, err := parsePURL(purl)
				if err != nil {
					return nil, fmt.Errorf(
						"%s: %w", id, err,
					)
				}

				key := id + "|" + purl
				if seen[key] {
					continue
				}
				seen[key] = true

				report.Vulnerabilities = append(
					report.Vulnerabilities,
					Vulnerability{
						ID:          id,
						Package:     name,
						Version:     version,
						FixVersion:  fix,
						Severity:    severity,
						Description: description,
						Links:       links,
					},
				)

				if severityRank(severity) > severityRank(report.Severity) {
					report.Severity = severity
				}
			}
		}
	}

	sort.Slice(report.Vulnerabilities, func(i, j int) bool {
		a := report.Vulnerabilities[i]
		b := report.Vulnerabilities[j]
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		return a.Version < b.Version
	})

	return report, nil
}

func parsePURL(purl string) (string, string, error) {
	if !strings.HasPrefix(purl, "pkg:") {
		return "", "", fmt.Errorf("invalid PURL %q", purl)
	}

	raw := strings.TrimPrefix(purl, "pkg:")
	raw = strings.SplitN(raw, "?", 2)[0]
	raw = strings.SplitN(raw, "#", 2)[0]

	slash := strings.IndexByte(raw, '/')
	if slash < 1 {
		return "", "", fmt.Errorf("invalid PURL %q", purl)
	}

	path := raw[slash+1:]
	at := strings.LastIndexByte(path, '@')
	if at < 1 || at == len(path)-1 {
		return "", "", fmt.Errorf("missing PURL version: %q", purl)
	}

	name, err := url.PathUnescape(path[:at])
	if err != nil {
		return "", "", err
	}
	version, err := url.PathUnescape(path[at+1:])
	if err != nil {
		return "", "", err
	}

	return name, version, nil
}

func normalizeSeverity(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical":
		return "Critical"
	case "high":
		return "High"
	case "medium", "moderate":
		return "Medium"
	case "low":
		return "Low"
	case "negligible", "informational", "info":
		return "Negligible"
	default:
		return "Unknown"
	}
}

func severityRank(s string) int {
	switch s {
	case "Critical":
		return 5
	case "High":
		return 4
	case "Medium":
		return 3
	case "Low":
		return 2
	case "Negligible":
		return 1
	default:
		return 0
	}
}
