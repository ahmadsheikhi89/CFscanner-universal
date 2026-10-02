package tui

import (
	"context"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Controller manages the TUI program and communication with the scanner
type Controller struct {
	program  *tea.Program
	model    *Model
	updateCh chan ScanUpdate
	mu       sync.RWMutex
}

// NewController creates a new TUI controller
func NewController(totalIPs int64, workerCount int) *Controller {
	model := NewModel(totalIPs, workerCount)

	ctrl := &Controller{
		model:    &model,
		updateCh: make(chan ScanUpdate, 100),
	}

	// Create program with AltScreen for proper screen clearing
	ctrl.program = tea.NewProgram(ctrl.model, tea.WithAltScreen())

	return ctrl
}

// Start starts the TUI program
func (c *Controller) Start() error {
	// Start message handlers
	go c.handleUpdates()

	// Run the program
	_, err := c.program.Run()
	return err
}

// SendUpdate sends a scan update to the TUI
func (c *Controller) SendUpdate(workerID int, ip string, success bool, errorMsg string) {
	select {
	case c.updateCh <- ScanUpdate{WorkerID: workerID, IP: ip, Success: success, ErrorMsg: errorMsg}:
	default:
		// Channel full, skip this update
	}
}

// SendRetryAttempt sends a retry attempt log without affecting progress counters
func (c *Controller) SendRetryAttempt(workerID int, ip string, errorMsg string) {
	if c.program != nil {
		c.program.Send(RetryAttempt{WorkerID: workerID, IP: ip, ErrorMsg: errorMsg})
	}
}

// SendWorkerUpdate sends a worker status update
func (c *Controller) SendWorkerUpdate(workerID int, ip string) {
	if c.program != nil {
		c.program.Send(WorkerUpdate{WorkerID: workerID, CurrentIP: ip})
	}
}

// SendStartupLog sends a startup log message to the TUI
func (c *Controller) SendStartupLog(message string) {
	if c.model != nil {
		c.model.AddStartupLog(message)
	}
}

// SetConfig sets a configuration key-value pair
func (c *Controller) SetConfig(key, value string) {
	if c.model != nil {
		c.model.SetConfig(key, value)
	}
}

// IsPaused returns whether the scanner is paused
func (c *Controller) IsPaused() bool {
	return c.model.IsPaused()
}

// WaitForResume blocks until the scanner is resumed
func (c *Controller) WaitForResume() {
	// Wait until not paused
	for c.IsPaused() {
		time.Sleep(100 * time.Millisecond)
	}
}

// GetContext returns the model's context for cancellation detection
func (c *Controller) GetContext() context.Context {
	return c.model.GetContext()
}

// GetStopChannel returns the stop channel
func (c *Controller) GetStopChannel() <-chan struct{} {
	return c.model.GetStopChannel()
}

// handleUpdates processes scan updates
func (c *Controller) handleUpdates() {
	for update := range c.updateCh {
		c.program.Send(update)
	}
}

// Quit signals the program to quit
func (c *Controller) Quit() {
	c.program.Quit()
}
