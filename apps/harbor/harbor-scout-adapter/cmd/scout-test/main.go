package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"harbor-scout-adapter/internal/converter"
	"harbor-scout-adapter/internal/scanner"
)

func main() {
	image := "harbor.prod.home.dblanchet.net/scdb/mysql-cluster:v4.1.0"

	artifact := &converter.Artifact{
		Repository: "scdb/mysql-cluster",
		Tag:        "v4.1.0",
	}

	scout := scanner.NewScout()

	log.Printf("Scanning %s", image)

	report, err := scout.Scan(
		context.Background(),
		image,
		artifact,
	)
	if err != nil {
		log.Fatal(err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(report); err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"Scan completed: %d vulnerabilities, severity=%s",
		len(report.Vulnerabilities),
		report.Severity,
	)
}
