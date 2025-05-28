package handlers

import (
	"fmt"
	"time"
)

const (
	// GameDuration defines the total duration of a game in minutes.
	GameDuration = 3 * time.Minute
)

// GameTimer holds the state of the game timer.
type GameTimer struct {
	startTime time.Time
	duration  time.Duration
}

// NewGameTimer creates and returns a new GameTimer instance.
// It initializes the timer with the predefined GameDuration.
func NewGameTimer() *GameTimer {
	return &GameTimer{
		duration: GameDuration,
	}
}

// Start records the current time as the beginning of the game.
func (gt *GameTimer) Start() {
	gt.startTime = time.Now()
}

// IsTimeUp checks if the game duration has elapsed since the timer started.
// It returns true if the current time is past the end time, false otherwise.
func (gt *GameTimer) IsTimeUp() bool {
	return time.Since(gt.startTime) >= gt.duration
}

// TimeRemaining calculates and returns the time left in the game.
// If the game has already ended (time is up), it returns 0.
func (gt *GameTimer) TimeRemaining() time.Duration {
	elapsed := time.Since(gt.startTime)
	if elapsed >= gt.duration {
		return 0
	}
	return gt.duration - elapsed
}

// FormattedTimeRemaining returns the remaining time as a formatted string (MM:SS).
func (gt *GameTimer) FormattedTimeRemaining() string {
	remaining := gt.TimeRemaining()
	if remaining <= 0 {
		return "00:00"
	}
	minutes := int(remaining.Minutes())
	seconds := int(remaining.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
