package report

import (
	"fmt"
	"strings"

	"biathlonPrototype/internal/domain"
)

// GenerateReport creates the final report as a slice of lines
func GenerateReport(competitors []*domain.Competitor) []string {
	reportLines := make([]string, 0, len(competitors))

	for _, competitor := range competitors {
		reportLines = append(reportLines, formatCompetitorResult(competitor))
	}

	return reportLines
}

// formatCompetitorResult formats the report string for a single competitor
func formatCompetitorResult(competitor *domain.Competitor) string {
	finalStatus := competitor.FinalStatusString()

	lapDetailsStr := formatLapDetails(competitor.LapDetails, competitor.Status, competitor.CurrentLap)
	penaltyDetailsStr := formatPenaltyDetails(competitor.PenaltyDetails, competitor.TotalPenaltyLaps > 0)
	shootingStr := fmt.Sprintf("%d/%d", competitor.TotalHits, competitor.TotalShots)

	return fmt.Sprintf("%s %d %s %s %s",
		finalStatus,
		competitor.ID,
		lapDetailsStr,
		penaltyDetailsStr,
		shootingStr,
	)
}

// formatLapDetails formats lap details
func formatLapDetails(lapDetails []domain.LapDetail, status domain.CompetitorStatus, currentLap int) string {
	var parts []string
	numLapsCompleted := len(lapDetails)
	totalExpectedLapEntries := numLapsCompleted

	if status == domain.StatusNotStarted {
		totalExpectedLapEntries = 0
	} else if status != domain.StatusFinished && currentLap > numLapsCompleted {
		totalExpectedLapEntries = currentLap
	}

	for i := 0; i < totalExpectedLapEntries; i++ {
		if i < len(lapDetails) && lapDetails[i].Duration > 0 {
			lapTimeStr := domain.FormatDuration(lapDetails[i].Duration)
			lapSpeedStr := fmt.Sprintf("%.3f", lapDetails[i].Speed)
			parts = append(parts, fmt.Sprintf("{%s, %s}", lapTimeStr, lapSpeedStr))
		} else {
			parts = append(parts, "{,}")
		}
	}

	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

// formatPenaltyDetails formats the penalty information
func formatPenaltyDetails(penalty domain.PenaltyDetail, hadPenalties bool) string {
	if !hadPenalties {
		return "{,}"
	}
	if penalty.TotalDuration <= 0 {
		return fmt.Sprintf("{%s, %.3f}", domain.FormatDuration(0), 0.0)
	}

	penaltyTimeStr := domain.FormatDuration(penalty.TotalDuration)
	penaltySpeedStr := fmt.Sprintf("%.3f", penalty.AverageSpeed)
	return fmt.Sprintf("{%s, %s}", penaltyTimeStr, penaltySpeedStr)
}
