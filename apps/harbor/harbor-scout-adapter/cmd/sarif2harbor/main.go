package main

import (
  "encoding/json"
  "flag"
  "fmt"
  "os"

  "harbor-scout-adapter/internal/converter"
)

func main() {
  input := flag.String("input", "", "Input SARIF file")
  output := flag.String("output", "", "Output Harbor JSON file")
  repository := flag.String("repository", "", "Artifact repository")
  digest := flag.String("digest", "", "Artifact digest")
  flag.Parse()

  if *input == "" || *output == "" {
    fmt.Fprintln(os.Stderr, "input and output are required")
    os.Exit(2)
  }

  file, err := os.Open(*input)
  if err != nil {
    fail(err)
  }
  defer file.Close()

  var artifact *converter.Artifact
  if *repository != "" || *digest != "" {
    artifact = &converter.Artifact{
      Repository: *repository,
      Digest: *digest,
    }
  }

  report, err := converter.Convert(file, artifact)
  if err != nil {
    fail(err)
  }

  out, err := os.Create(*output)
  if err != nil {
    fail(err)
  }
  defer out.Close()

  encoder := json.NewEncoder(out)
  encoder.SetIndent("", "  ")

  if err := encoder.Encode(report); err != nil {
    fail(err)
  }

  fmt.Printf(
    "Converted %d vulnerabilities; severity=%s\n",
    len(report.Vulnerabilities),
    report.Severity,
  )
}

func fail(err error) {
  fmt.Fprintln(os.Stderr, err)
  os.Exit(1)
}

