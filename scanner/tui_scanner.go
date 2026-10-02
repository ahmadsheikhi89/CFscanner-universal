package scanner

import (
	config "CFScanner/configuration"
	"CFScanner/tui"
	"fmt"
	"sync"
	"time"
)

// Global TUI controller and state
var tuiCtrl *tui.Controller
var tuiMu sync.Mutex

// SetTUIController sets the global TUI controller
func SetTUIController(ctrl *tui.Controller) {
	tuiMu.Lock()
	defer tuiMu.Unlock()
	tuiCtrl = ctrl
}

// UpdateTUI sends an update to the TUI
func UpdateTUI(workerID int, ip string, success bool) {
	tuiMu.Lock()
	ctrl := tuiCtrl
	tuiMu.Unlock()

	if ctrl != nil {
		ctrl.SendUpdate(workerID, ip, success, "")
	}
}

// UpdateTUIWithError sends an update with error message to the TUI
func UpdateTUIWithError(workerID int, ip string, success bool, errorMsg string) {
	tuiMu.Lock()
	ctrl := tuiCtrl
	tuiMu.Unlock()

	if ctrl != nil {
		ctrl.SendUpdate(workerID, ip, success, errorMsg)
	}
}

// UpdateTUIRetryAttempt sends a retry attempt log without affecting progress counters
func UpdateTUIRetryAttempt(workerID int, ip string, errorMsg string) {
	tuiMu.Lock()
	ctrl := tuiCtrl
	tuiMu.Unlock()

	if ctrl != nil {
		ctrl.SendRetryAttempt(workerID, ip, errorMsg)
	}
}

// UpdateWorker sends a worker status update to the TUI
func UpdateWorker(workerID int, ip string) {
	tuiMu.Lock()
	ctrl := tuiCtrl
	tuiMu.Unlock()

	if ctrl != nil {
		ctrl.SendWorkerUpdate(workerID, ip)
	}
}

// UpdateTUIWorkerStatus updates the worker status in TUI with custom message
func UpdateTUIWorkerStatus(workerID int, ip string, status string) {
	tuiMu.Lock()
	ctrl := tuiCtrl
	tuiMu.Unlock()

	if ctrl != nil {
		ctrl.SendWorkerUpdate(workerID, fmt.Sprintf("%s - %s", ip, status))
	}
}

// IsTUIPaused returns whether the TUI is paused
func IsTUIPaused() bool {
	tuiMu.Lock()
	ctrl := tuiCtrl
	tuiMu.Unlock()

	if ctrl != nil {
		return ctrl.IsPaused()
	}
	return false
}

// WaitTUIResume waits for the TUI to resume
func WaitTUIResume() {
	tuiMu.Lock()
	ctrl := tuiCtrl
	tuiMu.Unlock()

	if ctrl != nil {
		ctrl.WaitForResume()
	}
}

// GetTUIContext returns the TUI context for cancellation
func GetTUIContext() <-chan struct{} {
	tuiMu.Lock()
	ctrl := tuiCtrl
	tuiMu.Unlock()

	if ctrl != nil {
		return ctrl.GetStopChannel()
	}
	return nil
}

// IsTUIActive returns whether TUI is active
func IsTUIActive() bool {
	tuiMu.Lock()
	ctrl := tuiCtrl
	tuiMu.Unlock()

	return ctrl != nil
}

// StartWithTUI starts the scanner with TUI interface
func StartWithTUI(C config.Configuration, Worker config.Worker, ipList []string, threadsCount int, tuiController *tui.Controller) {
	// Set global TUI controller
	SetTUIController(tuiController)

	// Send startup info after a small delay to ensure TUI is ready
	go func() {
		time.Sleep(200 * time.Millisecond)
		tuiController.SendStartupLog(fmt.Sprintf("[Xray 26.1.23 (Xray, Penetrates Everything.) Custom (go1.25.6 linux/amd64) A unified platform for anti-censorship.]"))
		tuiController.SendStartupLog(fmt.Sprintf("Starting to scan %d IPS.", len(ipList)))
	}()

	// Start scanning in background
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		StartScanWithWorkerIDs(C, Worker, ipList, threadsCount)

		// Scanning completed, send completion message and auto-exit
		tuiController.SendStartupLog("")
		tuiController.SendStartupLog("✅ SCANNING COMPLETED!")
		tuiController.SendStartupLog("Results have been saved to the output file.")
		tuiController.SendStartupLog("Press Q to exit or wait 10 seconds for auto-exit...")

		// Auto-exit after 10 seconds
		go func() {
			time.Sleep(10 * time.Second)
			tuiController.Quit()
		}()
	}()

	// Start TUI in main thread (blocks until quit)
	_ = tuiController.Start()

	// TUI has exited, scanning should also stop
	// The scanner will check for stop signal in its loop
}
