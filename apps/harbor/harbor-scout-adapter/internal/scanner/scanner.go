package scanner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"harbor-scout-adapter/internal/converter"
)

type Status string

const (
	Pending   Status = "pending"
	Running   Status = "running"
	Completed Status = "completed"
	Failed    Status = "failed"
)

type Job struct {
	Status    Status
	Report    *converter.Report
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type request struct {
	id            string
	image         string
	artifact      converter.Artifact
	authorization string
}

type Service struct {
	mu        sync.RWMutex
	jobs      map[string]Job
	queue     chan request
	sarifPath string
	scout     *Scout
	ttl       time.Duration
}

func New(sarifPath string) *Service {
	return newService(sarifPath, nil)
}

func NewLive() *Service {
	return newService("", NewScout())
}

func newService(path string, scout *Scout) *Service {
	s := &Service{
		jobs:      make(map[string]Job),
		queue:     make(chan request, 16),
		sarifPath: path,
		scout:     scout,
		ttl:       time.Hour,
	}

	go s.worker()
	go s.cleanup()

	return s
}

func (s *Service) Submit(
	artifact converter.Artifact,
	image string,
	authorization string,
) (string, error) {
	if image == "" {
		return "", fmt.Errorf("image reference is empty")
	}

	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}

	id := hex.EncodeToString(token[:])
	now := time.Now()

	req := request{
		id:            id,
		image:         image,
		artifact:      artifact,
		authorization: authorization,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case s.queue <- req:
		s.jobs[id] = Job{
			Status:    Pending,
			CreatedAt: now,
			UpdatedAt: now,
		}
		return id, nil
	default:
		return "", fmt.Errorf("scan queue is full")
	}
}

func (s *Service) Get(id string) (Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	return job, ok
}

func (s *Service) update(id string, fn func(*Job)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return
	}

	fn(&job)
	job.UpdatedAt = time.Now()
	s.jobs[id] = job
}

func (s *Service) worker() {
	for req := range s.queue {
		s.update(req.id, func(job *Job) {
			job.Status = Running
		})

		report, err := s.execute(req)

		s.update(req.id, func(job *Job) {
			if err != nil {
				job.Status = Failed
				job.Error = err.Error()
				return
			}

			job.Status = Completed
			job.Report = report
		})
	}
}

func (s *Service) execute(
	req request,
) (*converter.Report, error) {
	if s.scout != nil {
		return s.scout.Scan(
			context.Background(),
			req.image,
			&req.artifact,
			req.authorization,
		)
	}

	file, err := os.Open(s.sarifPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return converter.Convert(file, &req.artifact)
}

func (s *Service) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-s.ttl)

		s.mu.Lock()
		for id, job := range s.jobs {
			if (job.Status == Completed ||
				job.Status == Failed) &&
				job.UpdatedAt.Before(cutoff) {
				delete(s.jobs, id)
			}
		}
		s.mu.Unlock()
	}
}

// ImageReference builds a registry image reference.
// Only use registry hosts authorized by the deployment.
func ImageReference(
	registryURL string,
	artifact converter.Artifact,
) (string, error) {
	host := strings.TrimPrefix(registryURL, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")

	if host == "" ||
		strings.ContainsAny(host, "/@ \t\n") {
		return "", fmt.Errorf("invalid registry host")
	}

	repository := artifact.Repository
	if repository == "" ||
		strings.HasPrefix(repository, "/") ||
		strings.Contains(repository, "..") ||
		strings.ContainsAny(repository, "@ \t\n") {
		return "", fmt.Errorf("invalid repository")
	}

	if artifact.Digest == "" {
		return "", fmt.Errorf("missing artifact digest")
	}

	if !strings.HasPrefix(artifact.Digest, "sha256:") ||
		len(artifact.Digest) != 71 {
		return "", fmt.Errorf("invalid SHA256 digest")
	}

	for _, c := range artifact.Digest[7:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return "", fmt.Errorf("invalid SHA256 digest")
		}
	}

	return host + "/" + repository + "@" +
		artifact.Digest, nil
}
