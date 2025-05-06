package processing

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"biathlonPrototype/internal/config"
	"biathlonPrototype/internal/domain"
)

// Simulator manages the state of the simulation
type Simulator struct {
	Config      *config.Config
	Competitors map[int]*domain.Competitor
	Events      []*domain.Event
	CurrentTime time.Time
	OutputLog   []string
}

// NewSimulator creates a new simulator
func NewSimulator(cfg *config.Config) *Simulator {
	return &Simulator{
		Config:      cfg,
		Competitors: make(map[int]*domain.Competitor),
		Events:      make([]*domain.Event, 0),
		OutputLog:   make([]string, 0),
	}
}

// LoadEventsFromFile loads and processes events from a file
func (simulator *Simulator) LoadEventsFromFile(filePath string) (err error) {
	var file *os.File
	file, err = os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening event file %s: %w", filePath, err)
	}
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("error closing event file %s: %w", filePath, closeErr)
		} else if closeErr != nil {
			fmt.Fprintf(os.Stderr, "Additional error while closing event file %s: %v (original error: %v)\n", filePath, closeErr, err)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	var previousTimestamp time.Time
	var processEventErr error

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event *domain.Event
		event, err = domain.ParseEventFromString(line)
		if err != nil {
			err = fmt.Errorf("string parsing error at line %d ('%s'): %w", lineNumber, line, err)
			return err
		}

		if !previousTimestamp.IsZero() && event.Timestamp.Before(previousTimestamp) {
			err = fmt.Errorf("time order of events on line %d is broken: %s before %s", lineNumber, domain.FormatTime(event.Timestamp), domain.FormatTime(previousTimestamp))
			return err
		}
		previousTimestamp = event.Timestamp

		processEventErr = simulator.ProcessEvent(event)
		if processEventErr != nil {
			err = fmt.Errorf("event handling error at line %d ('%s'): %w", lineNumber, line, processEventErr)
			return err
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		if err == nil {
			err = fmt.Errorf("error reading event file %s: %w", filePath, scanErr)
		} else {
			fmt.Fprintf(os.Stderr, "Additional error while scanning event file %s: %v (original error: %v)\n", filePath, scanErr, err)
		}
	}

	if err == nil {
		simulator.CheckForNotStarted()
	}

	return err
}

// ProcessEvent processes a single event and updates the simulation state
func (simulator *Simulator) ProcessEvent(event *domain.Event) error {
	simulator.CurrentTime = event.Timestamp
	simulator.Events = append(simulator.Events, event)

	if event.IsIncoming {
		simulator.OutputLog = append(simulator.OutputLog, event.String())
	}

	competitor, competitorExists := simulator.Competitors[event.CompetitorID]

	if event.ID == domain.Register {
		if competitorExists {
			fmt.Printf("Warning: Competitor %d re-registered in %s\n", event.CompetitorID, domain.FormatTime(event.Timestamp))
		} else {
			simulator.Competitors[event.CompetitorID] = domain.NewCompetitor(event.CompetitorID, event.Timestamp)
		}
		return nil
	}

	if !competitorExists {
		return fmt.Errorf("event ID %d for unregistered competitor %d", event.ID, event.CompetitorID)
	}

	if event.ID != domain.SetStartTime {
		competitor.LastEventTime = event.Timestamp
	}

	switch event.ID {
	case domain.SetStartTime:
		if len(event.ExtraParameters) < 1 {
			return fmt.Errorf("start time missing for SetStartTime event for competitor %d", competitor.ID)
		}
		scheduledTime, err := domain.ParseTimeFromString(fmt.Sprintf("[%s]", event.ExtraParameters[0]))
		if err != nil {
			return fmt.Errorf("invalid start time format '%s' for competitor %d: %v", event.ExtraParameters[0], competitor.ID, err)
		}
		competitor.ScheduledStartTime = scheduledTime

	case domain.OnStartLine:
		if competitor.Status == domain.StatusRegistered || competitor.Status == domain.StatusReadyToStart {
			competitor.Status = domain.StatusReadyToStart
		} else {
			fmt.Printf("Warning: OnStartLine event (%d) in unexpected status %s", competitor.ID, competitor.Status)
		}

	case domain.Started:
		if competitor.Status != domain.StatusReadyToStart && competitor.Status != domain.StatusRegistered {
			fmt.Printf("Warning: Event Started (%d) in unexpected status %s (expected ReadyToStart or Registered)\n", competitor.ID, competitor.Status)
		}
		if !competitor.ScheduledStartTime.IsZero() {
			startDeadline := competitor.ScheduledStartTime.Add(simulator.Config.ParsedStartDelta)
			if event.Timestamp.After(startDeadline) {
				simulator.DisqualifyCompetitor(competitor, event.Timestamp, "NotStarted")
				return nil
			}
		}
		competitor.Status = domain.StatusStarted
		competitor.ActualStartTime = event.Timestamp
		competitor.CurrentLap = 1
		competitor.CurrentLapStartTime = event.Timestamp

	case domain.EnterFiringRange:
		if competitor.TotalFiringRangesCompleted >= simulator.Config.FiringLines {
			msg := fmt.Sprintf("competitor %d attempts to enter the firing line after completing all %d required lines (completed: %d)",
				competitor.ID, simulator.Config.FiringLines, competitor.TotalFiringRangesCompleted)
			fmt.Printf("Warning: %s\n", msg)
			simulator.DisqualifyCompetitor(competitor, event.Timestamp, "Extra firing line")

			return nil
		}

		actualRangeNumFromEvent := 0
		if len(event.ExtraParameters) > 0 {
			rangeNum, err := strconv.Atoi(event.ExtraParameters[0])
			if err == nil {
				actualRangeNumFromEvent = rangeNum
			} else {
				return fmt.Errorf("invalid milestone number '%s' in event 5 for competitor %d", event.ExtraParameters[0], competitor.ID)
			}
		} else {
			return fmt.Errorf("missing milestone number in event 5 for competitor %d", competitor.ID)
		}

		expectedRangeGlobalNum := competitor.TotalFiringRangesCompleted + 1
		if actualRangeNumFromEvent != expectedRangeGlobalNum {
			fmt.Printf("Warning: competitor %d (ID %d) has reached milestone %d (by event), although the expected milestone was %d (completed: %d).\n",
				competitor.ID, competitor.ID, actualRangeNumFromEvent, expectedRangeGlobalNum, competitor.TotalFiringRangesCompleted)

			if actualRangeNumFromEvent <= 0 || actualRangeNumFromEvent > simulator.Config.FiringLines {
				return fmt.Errorf("invalid milestone number %d from event for competitor %d (max: %d)", actualRangeNumFromEvent, competitor.ID, simulator.Config.FiringLines)
			}
		}

		if actualRangeNumFromEvent > simulator.Config.FiringLines {
			return fmt.Errorf("competitor %d entered the %d milestone, which is more than the configured %d milestones for the race",
				competitor.ID, actualRangeNumFromEvent, simulator.Config.FiringLines)
		}

		if competitor.Status != domain.StatusStarted && competitor.Status != domain.StatusPenalized {
			fmt.Printf("Warning: EnterFiringRange event (%d) in unexpected status %s (expected Started or Penalized)\n", competitor.ID, competitor.Status)
		}

		competitor.Status = domain.StatusFiring
		competitor.HitsThisRange = 0
		competitor.LastFiringRangeEntered = actualRangeNumFromEvent

	case domain.HitTarget:
		if competitor.Status != domain.StatusFiring {
			fmt.Printf("Warning: HitTarget event (%d) out of range (status %s)\n", competitor.ID, competitor.Status)
		} else {
			competitor.HitsThisRange++
			competitor.TotalHits++
		}

	case domain.LeaveFiringRange:
		if competitor.Status != domain.StatusFiring {
			fmt.Printf("Warning: LeaveFiringRange event (%d) in unexpected status %s (expected Firing)\n", competitor.ID, competitor.Status)
			competitor.LastFiringRangeEntered = 0
			return nil
		}

		shotsThisRange := 0
		if competitor.LastFiringRangeEntered > 0 && competitor.LastFiringRangeEntered > competitor.TotalFiringRangesCompleted {
			shotsThisRange = 5
			competitor.TotalShots += shotsThisRange
			competitor.TotalFiringRangesCompleted++
		} else if competitor.LastFiringRangeEntered > 0 {
			fmt.Printf("Warning: competitor %d left range %d, which may have already been processed or wasn't expected (completed: %d).\n",
				competitor.ID, competitor.LastFiringRangeEntered, competitor.TotalFiringRangesCompleted)
		}

		misses := shotsThisRange - competitor.HitsThisRange
		if misses < 0 {
			fmt.Printf("Warning: competitor %d recorded %d hits with %d shots at range %d.\n",
				competitor.ID, competitor.HitsThisRange, shotsThisRange, competitor.LastFiringRangeEntered)
			misses = 0
		}
		competitor.MissesToPenalize += misses

		if competitor.MissesToPenalize == 0 {
			competitor.Status = domain.StatusStarted
		}

		competitor.HitsThisRange = 0
		competitor.LastFiringRangeEntered = 0

	case domain.EnterPenaltyLaps:
		if competitor.Status != domain.StatusFiring && competitor.Status != domain.StatusStarted && competitor.Status != domain.StatusPenalized {
			fmt.Printf("Warning: Event EnterPenaltyLaps (%d) in unexpected status %s\n", competitor.ID, competitor.Status)
		}
		if simulator.Config.PenaltyLen <= 0 {
			fmt.Printf("Warning: competitor %d entered the penalty laps, but their length is 0. Let's skip.\n", competitor.ID)
			competitor.Status = domain.StatusStarted
			return nil
		}
		if competitor.MissesToPenalize == 0 {
			fmt.Printf("Warning: competitor %d entered the penalty laps without any outstanding penalties. We'll let him through.\n", competitor.ID)
			competitor.Status = domain.StatusStarted
			return nil
		}
		competitor.Status = domain.StatusPenalized
		competitor.PenaltyStartTime = event.Timestamp

	case domain.LeavePenaltyLaps:
		if competitor.Status != domain.StatusPenalized {
			fmt.Printf("Warning: LeavePenaltyLaps event (%d) in unexpected status %s (expected Penalized)\n", competitor.ID, competitor.Status)
			if competitor.PenaltyStartTime.IsZero() {
				fmt.Printf("Warning: competitor %d (status %s) has no penalty lap entry time, LeavePenaltyLaps event ignored for time calculation.\n", competitor.ID, competitor.Status)
				return nil
			}
		}

		if competitor.PenaltyStartTime.IsZero() {
			fmt.Printf("Error/Warning: competitor %d left penalty laps but entry time (PenaltyStartTime) was not recorded. Cannot calculate penalty duration.\n", competitor.ID)
		} else {
			penaltyDuration := event.Timestamp.Sub(competitor.PenaltyStartTime)
			if penaltyDuration < 0 {
				fmt.Printf("Error/Warning: Negative penalty lap duration (%s) for competitor %d. Ignored.\n", domain.FormatDuration(penaltyDuration), competitor.ID)
			} else {
				competitor.TotalPenaltyTime += penaltyDuration
			}
			competitor.PenaltyStartTime = time.Time{}
		}

		competitor.TotalPenaltyLaps += competitor.MissesToPenalize
		competitor.MissesToPenalize = 0

		competitor.Status = domain.StatusStarted

	case domain.EndLap:
		if competitor.Status != domain.StatusStarted {
			fmt.Printf("Warning: EndLap event (%d) in unexpected status %s (expected Started)\n", competitor.ID, competitor.Status)
		}

		lapDuration := event.Timestamp.Sub(competitor.CurrentLapStartTime)
		lapSpeed := domain.CalculateSpeed(simulator.Config.LapLen, lapDuration)

		if len(competitor.LapDetails) < competitor.CurrentLap {
			competitor.LapDetails = append(competitor.LapDetails, make([]domain.LapDetail, competitor.CurrentLap-len(competitor.LapDetails))...)
		}
		competitor.LapDetails[competitor.CurrentLap-1] = domain.LapDetail{
			Duration: lapDuration,
			Speed:    lapSpeed,
		}

		if competitor.CurrentLap >= simulator.Config.Laps {
			if competitor.TotalFiringRangesCompleted < simulator.Config.FiringLines {
				reason := fmt.Sprintf("Not all %d firing ranges completed (completed %d)", simulator.Config.FiringLines, competitor.TotalFiringRangesCompleted)
				fmt.Printf("Warning/Error: competitor %d (ID %d) is finishing but %s.\n", competitor.ID, competitor.ID, reason)
			}
			simulator.FinishCompetitor(competitor, event.Timestamp)
		} else {
			competitor.CurrentLap++
			competitor.CurrentLapStartTime = event.Timestamp
		}

	case domain.CannotContinue:
		if competitor.Status != domain.StatusFinished && competitor.Status != domain.StatusNotStarted && competitor.Status != domain.StatusDisqualified {
			competitor.Status = domain.StatusNotFinished
			competitor.FinishTime = event.Timestamp
			reason := "Reason not specified"
			if len(event.ExtraParameters) > 0 {
				reason = strings.Join(event.ExtraParameters, " ")
			}
			competitor.DisqualificationReason = reason

			if competitor.TotalPenaltyLaps > 0 && simulator.Config.PenaltyLen > 0 {
				totalPenaltyDistance := float64(competitor.TotalPenaltyLaps) * simulator.Config.PenaltyLen
				avgPenaltySpeed := domain.CalculateSpeed(totalPenaltyDistance, competitor.TotalPenaltyTime)
				competitor.PenaltyDetails = domain.PenaltyDetail{
					TotalDuration: competitor.TotalPenaltyTime,
					AverageSpeed:  avgPenaltySpeed,
				}
			}
		} else {
			fmt.Printf("Warning: CannotContinue event (%d) for competitor in final status %s\n", competitor.ID, competitor.Status)
		}
	default:
		fmt.Printf("Warning: Unknown incoming event ID %d for competitor %d\n", event.ID, competitor.ID)
	}

	return nil
}

// FinishCompetitor handles the competitor's finish
func (simulator *Simulator) FinishCompetitor(competitor *domain.Competitor, finishTime time.Time) {
	if competitor.Status == domain.StatusFinished || competitor.Status == domain.StatusNotFinished || competitor.Status == domain.StatusNotStarted || competitor.Status == domain.StatusDisqualified {
		return
	}

	competitor.Status = domain.StatusFinished
	competitor.FinishTime = finishTime

	if competitor.TotalPenaltyLaps > 0 && simulator.Config.PenaltyLen > 0 {
		totalPenaltyDistance := float64(competitor.TotalPenaltyLaps) * simulator.Config.PenaltyLen
		avgPenaltySpeed := domain.CalculateSpeed(totalPenaltyDistance, competitor.TotalPenaltyTime)
		competitor.PenaltyDetails = domain.PenaltyDetail{
			TotalDuration: competitor.TotalPenaltyTime,
			AverageSpeed:  avgPenaltySpeed,
		}
	}

	finishEvent := &domain.Event{
		Timestamp:    finishTime,
		ID:           domain.Finished,
		CompetitorID: competitor.ID,
		IsIncoming:   false,
	}
	simulator.Events = append(simulator.Events, finishEvent)
	simulator.OutputLog = append(simulator.OutputLog, finishEvent.String())
}

// DisqualifyCompetitor handles competitor disqualification
func (simulator *Simulator) DisqualifyCompetitor(competitor *domain.Competitor, dqTime time.Time, reason string) {
	if competitor.Status == domain.StatusFinished || competitor.Status == domain.StatusNotFinished || competitor.Status == domain.StatusNotStarted || competitor.Status == domain.StatusDisqualified {
		return
	}

	competitor.FinishTime = dqTime

	if reason == "NotStarted" {
		competitor.Status = domain.StatusNotStarted
	} else {
		competitor.Status = domain.StatusDisqualified
	}
	competitor.DisqualificationReason = reason

	dqEvent := &domain.Event{
		Timestamp:       dqTime,
		ID:              domain.Disqualified,
		CompetitorID:    competitor.ID,
		ExtraParameters: []string{reason},
		IsIncoming:      false,
	}

	alreadyDQEventExists := false
	for _, event := range simulator.Events {
		if !event.IsIncoming && event.ID == domain.Disqualified && event.CompetitorID == competitor.ID {
			alreadyDQEventExists = true
			break
		}
	}

	if !alreadyDQEventExists {
		simulator.Events = append(simulator.Events, dqEvent)
		simulator.OutputLog = append(simulator.OutputLog, dqEvent.String())
	}
}

// CheckForNotStarted checks for athletes who were supposed to start but did not do so on time
func (simulator *Simulator) CheckForNotStarted() {
	for _, competitor := range simulator.Competitors {
		if (competitor.Status == domain.StatusRegistered || competitor.Status == domain.StatusReadyToStart) &&
			competitor.ActualStartTime.IsZero() &&
			!competitor.ScheduledStartTime.IsZero() {

			startDeadline := competitor.ScheduledStartTime.Add(simulator.Config.ParsedStartDelta)

			if simulator.CurrentTime.After(startDeadline) {
				if competitor.Status != domain.StatusNotFinished && competitor.Status != domain.StatusDisqualified {
					fmt.Printf("Info: competitor %d (ID %d) did not start by %s (deadline %s). Status: NotStarted.\n",
						competitor.ID, competitor.ID, domain.FormatTime(simulator.CurrentTime), domain.FormatTime(startDeadline))
					simulator.DisqualifyCompetitor(competitor, startDeadline, "NotStarted")
				}
			}
		}
	}
}

// GetSortedCompetitors returns a sorted list of athletes for the report
func (simulator *Simulator) GetSortedCompetitors() []*domain.Competitor {
	competitorsList := make([]*domain.Competitor, 0, len(simulator.Competitors))
	for _, c := range simulator.Competitors {
		competitorsList = append(competitorsList, c)
	}

	sort.SliceStable(competitorsList, func(i, j int) bool {
		c1 := competitorsList[i]
		c2 := competitorsList[j]

		c1IsFinished := c1.Status == domain.StatusFinished
		c2IsFinished := c2.Status == domain.StatusFinished

		if c1IsFinished && !c2IsFinished {
			return true
		}
		if !c1IsFinished && c2IsFinished {
			return false
		}

		if c1IsFinished && c2IsFinished {
			t1, ok1 := c1.CalculateTotalTime()
			t2, ok2 := c2.CalculateTotalTime()

			if ok1 && ok2 {
				if t1 != t2 {
					return t1 < t2
				}
				return c1.ID < c2.ID
			}
			if ok1 && !ok2 {
				return true
			}
			if !ok1 && ok2 {
				return false
			}
			return c1.ID < c2.ID
		}

		statusRank := func(status domain.CompetitorStatus) int {
			switch status {
			case domain.StatusNotFinished:
				return 1
			case domain.StatusNotStarted:
				return 2
			case domain.StatusDisqualified:
				return 3
			default:
				return 4
			}
		}

		rank1 := statusRank(c1.Status)
		rank2 := statusRank(c2.Status)

		if rank1 != rank2 {
			return rank1 < rank2
		}

		return c1.ID < c2.ID
	})

	return competitorsList
}
