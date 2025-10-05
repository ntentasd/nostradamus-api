package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ntentasd/nostradamus-api/internal/arroyo"
	"github.com/ntentasd/nostradamus-api/pkg/types"
)

// Supervisor checks Arroyo pipelines periodically and restarts failed ones.
type Supervisor struct {
	AC        *arroyo.ArroyoClient
	Interval  time.Duration
	cancelCtx context.CancelFunc
}

// NewSupervisor creates a new background worker for pipeline supervision.
func NewSupervisor(ac *arroyo.ArroyoClient, interval time.Duration) *Supervisor {
	return &Supervisor{
		AC:       ac,
		Interval: interval,
	}
}

func (s *Supervisor) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancelCtx = cancel

	go func() {
		ticker := time.NewTicker(s.Interval)
		defer ticker.Stop()

		log.Println("[supervisor] started pipeline monitor")

		for {
			select {
			case <-ctx.Done():
				log.Println("[supervisor] stopped")
				return
			case <-ticker.C:
				if err := s.checkAndRestartPipelines(); err != nil {
					log.Printf("[supervisor] error: %v", err)
				}
			}
		}
	}()
}

// Stop gracefully stops the background worker.
func (s *Supervisor) Stop() {
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
}

// checkAndRestartPipelines calls Arroyo's API to detect failed jobs.
func (s *Supervisor) checkAndRestartPipelines() error {
	// list pipelines
	resp, err := s.AC.Get("/api/v1/pipelines")
	if err != nil {
		return fmt.Errorf("failed to fetch pipelines: %w", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("failed to decode pipelines: %w", err)
	}

	// iterate each pipeline, inspect jobs
	for _, p := range payload.Data {
		jobResp, err := s.AC.Get(fmt.Sprintf("/api/v1/pipelines/%s/jobs", p.ID))
		if err != nil {
			log.Printf("[supervisor] failed to fetch jobs for pipeline %s: %v", p.ID, err)
			continue
		}

		var jobs struct {
			Data []struct {
				ID     string          `json:"id"`
				State  types.StateType `json:"state"`
				Failed string          `json:"failureMessage"`
			} `json:"data"`
		}
		if err := json.NewDecoder(jobResp.Body).Decode(&jobs); err != nil {
			log.Printf("[supervisor] decode error for %s: %v", p.ID, err)
			jobResp.Body.Close()
			continue
		}
		jobResp.Body.Close()

		// detect non-running pipelines
		hasFailed := false
		for _, j := range jobs.Data {
			if j.State == types.StateTypeStopped || j.State == types.StateTypeFailed {
				log.Printf("[supervisor] pipeline %s (%s) has failed job %s - reason: %s",
					p.Name, p.ID, j.ID, j.Failed)
				hasFailed = true
				break
			}
		}

		// restart pipeline if needed
		if hasFailed {
			log.Printf("[supervisor] restarting pipeline %s...", p.ID)
			if err := s.AC.RestartPipeline(p.ID); err != nil {
				log.Printf("[supervisor] restart failed for %s: %v", p.ID, err)
			} else {
				log.Printf("[supervisor] pipeline %s restarted successfully", p.ID)
			}
		}
	}

	return nil
}
