package tools

import (
	"context"
	"fmt"
	"time"
)

// GetCurrentTimeInput represents the input parameters for the GetCurrentTime function.
//
// swagger:model GetCurrentTimeInput
type GetCurrentTimeInput struct {
	// Format is the time format string according to Go's time formatting conventions.
	//
	// required: false
	// example: "2006-01-02T15:04:05Z07:00"
	Format string `json:"format,omitempty" jsonschema_description:"Time format string according to Go's time formatting conventions, default format is : 2006-01-02T15:04:05Z07:00"  jsonschema:"required"`

	// Location is the IANA time zone identifier.
	//
	// required: false
	// example: "UTC"
	Location string `json:"location,omitempty" jsonschema_description:"IANA time zone identifier (e.g., 'Asia/Colombo', 'America/New_York')" jsonschema:"required"`
}

// GetCurrentTimeOutput represents the output of the GetCurrentTime function.
type GetCurrentTimeOutput struct {
	// CurrentTime is the current time formatted as per input parameters.
	CurrentTime string `json:"currentTime" jsonschema_description:"Current time formatted as per input parameters"`
}

// GetCurrentTime retrieves the current time, formatted according to the input parameters.
func GetCurrentTime(ctx context.Context, input GetCurrentTimeInput) (GetCurrentTimeOutput, error) {
	// Set default format if not provided
	format := input.Format
	if format == "" {
		format = time.RFC3339
	}

	// Set default location to UTC if not provided
	loc := time.UTC
	if input.Location != "" {
		var err error
		loc, err = time.LoadLocation(input.Location)
		if err != nil {
			return GetCurrentTimeOutput{}, fmt.Errorf("invalid location: %v", err)
		}
	}

	// Get the current time in the specified location
	now := time.Now().In(loc)

	// Format the time
	currentTime := now.Format(format)

	return GetCurrentTimeOutput{
		CurrentTime: currentTime,
	}, nil
}
