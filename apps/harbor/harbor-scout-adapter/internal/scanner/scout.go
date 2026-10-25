package scanner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"harbor-scout-adapter/internal/converter"
)

type Scout struct {
	Timeout time.Duration
}

func NewScout() *Scout {
	return &Scout{
		Timeout: 10 * time.Minute,
	}
}

func (s *Scout) Scan(
	parent context.Context,
	image string,
	artifact *converter.Artifact,
) (*converter.Report, error) {

	if image == "" {
		return nil, fmt.Errorf("image reference is empty")
	}

	ctx, cancel := context.WithTimeout(parent, s.Timeout)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "harbor-scout-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	sarifPath := filepath.Join(tmpDir, "report.sarif")

	cmd := exec.CommandContext(
		ctx,
		"docker",
		"scout",
		"cves",
		"--format", "sarif",
		"--output", sarifPath,
		"registry://"+image,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("Docker Scout timeout or cancellation: %w", ctx.Err())
		}
		return nil, fmt.Errorf(
			"Docker Scout failed: %w: %s",
			err,
			stderr.String(),
		)
	}

	file, err := os.Open(sarifPath)
	if err != nil {
		return nil, fmt.Errorf("opening SARIF report: %w", err)
	}
	defer file.Close()

	report, err := converter.Convert(file, artifact)
	if err != nil {
		return nil, fmt.Errorf("converting SARIF: %w", err)
	}

	return report, nil
}
