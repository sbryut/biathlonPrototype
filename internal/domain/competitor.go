package domain

import (
	"fmt"
	"time"
)

// CompetitorStatus represents the competitor's status
type CompetitorStatus string

const (
	StatusRegistered   CompetitorStatus = "Registered"
	StatusReadyToStart CompetitorStatus = "ReadyToStart"
	StatusStarted      CompetitorStatus = "Started"
	StatusFiring       CompetitorStatus = "Firing"
	StatusPenalized    CompetitorStatus = "Penalized"
	StatusFinished     CompetitorStatus = "Finished"
	StatusNotFinished  CompetitorStatus = "NotFinished"
	StatusNotStarted   CompetitorStatus = "NotStarted"
	StatusDisqualified CompetitorStatus = "Disqualified"
)

// LapDetail stores information about the passage of the main lap
type LapDetail struct {
	Duration time.Duration
	Speed    float64
}

// PenaltyDetail stores information about penalty laps
type PenaltyDetail struct {
	TotalDuration time.Duration
	AverageSpeed  float64
}

// Competitor represents the athlete's state
type Competitor struct {
	ID                  int
	Status              CompetitorStatus
	ScheduledStartTime  time.Time
	ActualStartTime     time.Time
	FinishTime          time.Time
	LastEventTime       time.Time
	CurrentLap          int
	CurrentLapStartTime time.Time
	LapDetails          []LapDetail

	// Shooting
	LastFiringRangeEntered     int
	TotalFiringRangesCompleted int
	HitsThisRange              int
	TotalHits                  int
	TotalShots                 int

	// Fines
	MissesToPenalize       int
	PenaltyStartTime       time.Time
	TotalPenaltyTime       time.Duration
	TotalPenaltyLaps       int
	PenaltyDetails         PenaltyDetail
	DisqualificationReason string
}

// NewCompetitor creates a new athlete
func NewCompetitor(id int, registrationTime time.Time) *Competitor {
	return &Competitor{
		ID:            id,
		Status:        StatusRegistered,
		LastEventTime: registrationTime,
		LapDetails:    make([]LapDetail, 0),
	}
}

// CalculateTotalTime calculates the total time of the race
func (competitor *Competitor) CalculateTotalTime() (time.Duration, bool) {
	if competitor.Status != StatusFinished {
		return 0, false
	}
	if competitor.ActualStartTime.IsZero() {
		return 0, false
	}

	startDiff := competitor.ActualStartTime.Sub(competitor.ScheduledStartTime)
	if startDiff < 0 {
		startDiff = 0
	}
	raceDuration := competitor.FinishTime.Sub(competitor.ActualStartTime)

	return raceDuration + startDiff, true
}

// FinalStatusString returns a string representation of the final status
func (competitor *Competitor) FinalStatusString() string {
	switch competitor.Status {
	case StatusFinished:
		totalTime, ok := competitor.CalculateTotalTime()
		if ok {
			return FormatDuration(totalTime)
		}
		return "[Error Calculating Time]"
	case StatusNotFinished:
		return "[NotFinished]"
	case StatusNotStarted:
		return "[NotStarted]"
	case StatusDisqualified:
		reason := ""
		if competitor.DisqualificationReason != "" {
			reason = ": " + competitor.DisqualificationReason
		}
		return fmt.Sprintf("[Disqualified%s]", reason)
	default:
		return "[In Progress]"
	}
}
