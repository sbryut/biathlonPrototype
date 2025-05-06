package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// EventID type for event identifiers
type EventID int

const (
	Register         EventID = 1
	SetStartTime     EventID = 2
	OnStartLine      EventID = 3
	Started          EventID = 4
	EnterFiringRange EventID = 5
	HitTarget        EventID = 6
	LeaveFiringRange EventID = 7
	EnterPenaltyLaps EventID = 8
	LeavePenaltyLaps EventID = 9
	EndLap           EventID = 10
	CannotContinue   EventID = 11

	Disqualified EventID = 32
	Finished     EventID = 33
)

// Event structure to represent an event
type Event struct {
	Timestamp       time.Time
	ID              EventID
	CompetitorID    int
	ExtraParameters []string
	RawLine         string
	IsIncoming      bool
}

// ParseEventFromString parses an event from a log string
func ParseEventFromString(line string) (*Event, error) {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid event string format: %s", line)
	}

	timestamp, err := ParseTimeFromString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("error parsing time in string '%s': %v", line, err)
	}

	eventIDInt, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error parsing event ID in string '%s': %v", line, err)
	}
	eventID := EventID(eventIDInt)

	competitorID, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("error parsing athlete ID in line '%s': %v", line, err)
	}

	extraParameters := parts[3:]

	isIncoming := eventID >= Register && eventID <= CannotContinue

	return &Event{
		Timestamp:       timestamp,
		ID:              eventID,
		CompetitorID:    competitorID,
		ExtraParameters: extraParameters,
		RawLine:         line,
		IsIncoming:      isIncoming,
	}, nil
}

// String returns a string representation of the event for the log
func (event *Event) String() string {
	competitorStr := fmt.Sprintf("competitor(%d)", event.CompetitorID)
	var details string

	switch event.ID {
	case Register:
		details = fmt.Sprintf("The %s registered", competitorStr)
	case SetStartTime:
		startTimeStr := "N/A"
		if len(event.ExtraParameters) > 0 {
			parsedTime, err := time.Parse(TimeLayout, event.ExtraParameters[0])
			if err == nil {
				startTimeStr = parsedTime.Format(TimeLayout)
			} else {
				fmt.Printf("Warning: Unable to parse start time '%s' from SetStartTime event for output. Error: %v\n", event.ExtraParameters[0], err)
				startTimeStr = event.ExtraParameters[0]
			}
		}
		details = fmt.Sprintf("The start time for the %s was set by a draw to %s", competitorStr, startTimeStr)
	case OnStartLine:
		details = fmt.Sprintf("The %s is on the start line", competitorStr)
	case Started:
		details = fmt.Sprintf("The %s has started", competitorStr)
	case EnterFiringRange:
		rangeNum := "?"
		if len(event.ExtraParameters) > 0 {
			rangeNum = event.ExtraParameters[0]
		}
		details = fmt.Sprintf("The %s is on the firing range(%s)", competitorStr, rangeNum)
	case HitTarget:
		targetNum := "?"
		if len(event.ExtraParameters) > 0 {
			targetNum = event.ExtraParameters[0]
		}
		details = fmt.Sprintf("The target(%s) has been hit by %s", targetNum, competitorStr)
	case LeaveFiringRange:
		details = fmt.Sprintf("The %s left the firing range", competitorStr)
	case EnterPenaltyLaps:
		details = fmt.Sprintf("The %s entered the penalty laps", competitorStr)
	case LeavePenaltyLaps:
		details = fmt.Sprintf("The %s left the penalty laps", competitorStr)
	case EndLap:
		details = fmt.Sprintf("The %s ended the main lap", competitorStr)
	case CannotContinue:
		comment := ""
		if len(event.ExtraParameters) > 0 {
			comment = ": " + strings.Join(event.ExtraParameters, " ")
		}
		details = fmt.Sprintf("The %s can`t continue%s", competitorStr, comment)
	case Disqualified:
		reason := "Reason not specified"
		if len(event.ExtraParameters) > 0 {
			reason = strings.Join(event.ExtraParameters, " ")
		}
		details = fmt.Sprintf("The %s is disqualified (%s)", competitorStr, reason)
	case Finished:
		details = fmt.Sprintf("The %s has finished", competitorStr)
	default:
		details = fmt.Sprintf("Unknown event ID(%d) for %s", event.ID, competitorStr)
	}

	return fmt.Sprintf("%s %s", FormatTime(event.Timestamp), details)
}
