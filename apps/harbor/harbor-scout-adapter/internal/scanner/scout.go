package scanner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"harbor-scout-adapter/internal/converter"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
	authorization string,
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

	// Configuration Docker de base.
	// Elle peut provenir d'un Secret Kubernetes ou d'un fichier local.
	config := struct {
		Auths       map[string]map[string]string `json:"auths"`
		CredsStore  string                       `json:"credsStore,omitempty"`
		CredHelpers map[string]string            `json:"credHelpers,omitempty"`
	}{
		Auths: make(map[string]map[string]string),
	}

	baseConfigPath := os.Getenv("SCOUT_DOCKER_CONFIG")

	if baseConfigPath != "" {
		data, err := os.ReadFile(baseConfigPath)
		if err != nil {
			return nil, fmt.Errorf("reading Docker Scout config: %w", err)
		}

		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("invalid Docker Scout config: %w", err)
		}
	}

	if config.Auths == nil {
		config.Auths = make(map[string]map[string]string)
	}

	// Ajouter les identifiants Harbor pour cette analyse.
	if authorization != "" {
		const prefix = "Basic "

		if !strings.HasPrefix(authorization, prefix) {
			return nil, fmt.Errorf("unsupported registry authorization scheme")
		}

		credentials, err := base64.StdEncoding.DecodeString(
			strings.TrimSpace(strings.TrimPrefix(authorization, prefix)),
		)
		if err != nil {
			return nil, fmt.Errorf("invalid Basic authorization")
		}

		if !strings.Contains(string(credentials), ":") {
			return nil, fmt.Errorf("invalid Basic credentials")
		}

		registry := strings.SplitN(image, "/", 2)[0]

		config.Auths[registry] = map[string]string{
			"auth": base64.StdEncoding.EncodeToString(credentials),
		}
	}

	// Écrire la configuration temporaire.
	// Chaque analyse possède son propre répertoire.
	if baseConfigPath != "" || authorization != "" {
		data, err := json.Marshal(config)
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(
			filepath.Join(tmpDir, "config.json"),
			data,
			0600,
		); err != nil {
			return nil, fmt.Errorf("writing Docker credentials: %w", err)
		}
	}

	cmd := exec.CommandContext(
		ctx,
		"docker",
		"scout",
		"cves",
		"--format", "sarif",
		"--output", sarifPath,
		"registry://"+image,
	)

	// Utiliser la configuration temporaire lorsqu'elle existe.
	if baseConfigPath != "" || authorization != "" {
		cmd.Env = append(
			os.Environ(),
			"DOCKER_CONFIG="+tmpDir,
		)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf(
				"Docker Scout timeout or cancellation: %w",
				ctx.Err(),
			)
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
