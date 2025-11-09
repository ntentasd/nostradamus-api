package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ntentasd/nostradamus-api/internal/arroyo"
	"github.com/ntentasd/nostradamus-api/pkg/types"
	"github.com/rs/zerolog"
)

// Supervisor checks Arroyo pipelines periodically and restarts failed ones.
type Supervisor struct {
	AC        *arroyo.ArroyoClient
	Interval  time.Duration
	cancelCtx context.CancelFunc
	logger    zerolog.Logger
}

// NewSupervisor creates a new background worker for pipeline supervision.
func NewSupervisor(ac *arroyo.ArroyoClient, interval time.Duration, logger zerolog.Logger) *Supervisor {
	return &Supervisor{
		AC:       ac,
		Interval: interval,
		logger:   logger,
	}
}

func (s *Supervisor) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancelCtx = cancel

	go func() {
		ticker := time.NewTicker(s.Interval)
		defer ticker.Stop()

		s.logger.Info().Msg("pipeline monitoring started")

		for {
			select {
			case <-ctx.Done():
				s.logger.Info().Msg("pipeline monitoring stopped")
				return
			case <-ticker.C:
				if err := s.checkAndRestartPipelines(); err != nil {
					s.logger.Warn().Msg("failed to check and restart pipelines")
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
		s.logger.Error().Err(err).Msg("failed to fetch pipelines")
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
		s.logger.Error().Err(err).Msg("failed to decode pipelines")
		return fmt.Errorf("failed to decode pipelines: %w", err)
	}

	// iterate each pipeline, inspect jobs
	for _, p := range payload.Data {
		jobResp, err := s.AC.Get(fmt.Sprintf("/api/v1/pipelines/%s/jobs", p.ID))
		if err != nil {
			s.logger.Warn().Err(err).Str("pipeline_id", p.ID).Msg("failed to fetch jobs")
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
			s.logger.Warn().Err(err).Str("pipeline_id", p.ID).Msg("failed to decode pipeline")
			jobResp.Body.Close()
			continue
		}
		jobResp.Body.Close()

		// detect non-running pipelines
		hasFailed := false
		for _, j := range jobs.Data {
			if j.State == types.StateTypeStopped || j.State == types.StateTypeFailed {
				s.logger.Error().Str("pipeline_name", p.Name).Str("pipeline_id", p.ID).Str("job_id", j.ID).Str("reason", j.Failed).Msg("pipeline failed a job")
				hasFailed = true
				break
			}
		}

		// restart pipeline if needed
		if hasFailed {
			s.logger.Info().Str("pipeline_id", p.ID).Msg("restarting pipeline")
			if err := s.AC.RestartPipeline(p.ID); err != nil {
				s.logger.Error().Err(err).Str("pipeline_id", p.ID).Msg("failed to restart pipeline")
			} else {
				s.logger.Info().Str("pipeline_id", p.ID).Msg("pipeline restarted successfully")
			}
		}
	}

	return nil
}
