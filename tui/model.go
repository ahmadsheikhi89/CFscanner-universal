package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WorkerInfo holds information about a single worker
type WorkerInfo struct {
	ID        int
	CurrentIP string
	Active    bool
	LastSeen  time.Time
}

// Stats holds scanner statistics
type Stats struct {
	TotalIPs     int64
	ScannedIPs   int64
	SuccessIPs   int64
	FailedIPs    int64
	StartTime    time.Time
	IsPaused     bool
	Workers      map[int]*WorkerInfo
	TotalWorkers int
}

// ScanUpdate is a message for updating stats
type ScanUpdate struct {
	WorkerID int
	IP       string
	Success  bool
	ErrorMsg string // Error message for failed scans
}

// RetryAttempt is a message for logging retry attempts without affecting progress
type RetryAttempt struct {
	WorkerID int
	IP       string
	ErrorMsg string
}

// WorkerUpdate is a message for updating worker status
type WorkerUpdate struct {
	WorkerID  int
	CurrentIP string
}

// PauseToggle is a message for pause/unpause
type PauseToggle struct{}

// TickMsg is sent on each tick
type TickMsg time.Time

// Model represents the TUI state
type Model struct {
	stats    Stats
	mu       sync.RWMutex
	width    int
	height   int
	logs     []LogEntry
	maxLogs  int
	quitting bool
	stopCh   chan struct{}
	ctx      context.Context
	cancel   context.CancelFunc
	config   map[string]string
}

// LogEntry represents a log message
type LogEntry struct {
	Timestamp time.Time
	IP        string
	Message   string
	Type      LogType
}

type LogType int

const (
	LogInfo LogType = iota
	LogSuccess
	LogFail
	LogError
	LogStartup
)

// NewModel creates a new TUI model
func NewModel(totalIPs int64, workerCount int) Model {
	workers := make(map[int]*WorkerInfo)
	for i := 0; i < workerCount; i++ {
		workers[i] = &WorkerInfo{
			ID:       i,
			Active:   true,
			LastSeen: time.Now(),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	return Model{
		stats: Stats{
			TotalIPs:     totalIPs,
			StartTime:    time.Now(),
			IsPaused:     false,
			Workers:      workers,
			TotalWorkers: workerCount,
		},
		maxLogs: 15,
		logs:    make([]LogEntry, 0),
		stopCh:  make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
		config:  make(map[string]string),
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
	)
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			m.cancel()
			close(m.stopCh)
			go func() {
				time.Sleep(100 * time.Millisecond)
				os.Exit(0)
			}()
			return m, tea.Quit
		case "p", "P":
			return m, m.togglePause()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case ScanUpdate:
		m.mu.Lock()
		m.stats.ScannedIPs++

		// Update worker status
		if worker, exists := m.stats.Workers[msg.WorkerID]; exists {
			worker.CurrentIP = msg.IP
			worker.LastSeen = time.Now()
		}

		if msg.Success {
			m.stats.SuccessIPs++
			m.addLog(LogEntry{
				Timestamp: time.Now(),
				IP:        msg.IP,
				Message:   "✓ Success",
				Type:      LogSuccess,
			})
		} else {
			m.stats.FailedIPs++
			failMessage := "✗ Failed"
			if msg.ErrorMsg != "" {
				failMessage = fmt.Sprintf("✗ %s", msg.ErrorMsg)
			}
			m.addLog(LogEntry{
				Timestamp: time.Now(),
				IP:        msg.IP,
				Message:   failMessage,
				Type:      LogFail,
			})
		}
		m.mu.Unlock()

	case RetryAttempt:
		// Log retry attempt without affecting scan progress counters
		m.mu.Lock()
		failMessage := "✗ Retry attempt failed"
		if msg.ErrorMsg != "" {
			failMessage = fmt.Sprintf("✗ %s", msg.ErrorMsg)
		}
		m.addLog(LogEntry{
			Timestamp: time.Now(),
			IP:        msg.IP,
			Message:   failMessage,
			Type:      LogFail,
		})
		m.mu.Unlock()

	case WorkerUpdate:
		m.mu.Lock()
		if worker, exists := m.stats.Workers[msg.WorkerID]; exists {
			worker.CurrentIP = msg.CurrentIP
			worker.LastSeen = time.Now()
			worker.Active = true
		}
		m.mu.Unlock()

	case PauseToggle:
		m.mu.Lock()
		m.stats.IsPaused = !m.stats.IsPaused
		status := "Resumed"
		if m.stats.IsPaused {
			status = "Paused"
		}
		m.addLog(LogEntry{
			Timestamp: time.Now(),
			IP:        "",
			Message:   fmt.Sprintf("Scanner %s", status),
			Type:      LogInfo,
		})
		m.mu.Unlock()
		return m, nil

	case TickMsg:
		return m, tickCmd()
	}

	return m, nil
}

// AddStartupLog adds a startup configuration log entry
func (m *Model) AddStartupLog(message string) {
	m.mu.Lock()
	m.addLog(LogEntry{
		Timestamp: time.Now(),
		IP:        "",
		Message:   message,
		Type:      LogStartup,
	})
	m.mu.Unlock()
}

// SetConfig sets a configuration key-value pair
func (m *Model) SetConfig(key, value string) {
	m.mu.Lock()
	m.config[key] = value
	m.mu.Unlock()
}

// View renders the UI
func (m *Model) View() string {
	if m.quitting {
		return ""
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Colors
	purple := lipgloss.Color("135")
	green := lipgloss.Color("76")
	red := lipgloss.Color("196")
	yellow := lipgloss.Color("226")
	cyan := lipgloss.Color("51")
	gray := lipgloss.Color("244")

	// Styles
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(purple).
		MarginBottom(1).
		Render("⚡ CDN IP SCANNER - CLEAN IP DISCOVERY TOOL")

	// Configuration section
	configSection := ""
	if len(m.config) > 0 {
		configHeader := lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true).
			Render("📋 CONFIGURATION")

		// Create compact side-by-side layout
		leftColumn := []string{}
		rightColumn := []string{}

		// Define order and group items
		leftItems := []string{"host", "path", "port", "userid", "threads", "tries"}
		rightItems := []string{"download_speed", "upload_speed", "upload_test", "fronting_test", "xray_core", "writer"}

		for _, key := range leftItems {
			if value, exists := m.config[key]; exists {
				leftColumn = append(leftColumn, fmt.Sprintf("%s %-12s: %s",
					lipgloss.NewStyle().Foreground(green).Render("●"),
					lipgloss.NewStyle().Foreground(gray).Render(key),
					lipgloss.NewStyle().Foreground(purple).Render(value)))
			}
		}

		for _, key := range rightItems {
			if value, exists := m.config[key]; exists {
				rightColumn = append(rightColumn, fmt.Sprintf("%s %-12s: %s",
					lipgloss.NewStyle().Foreground(green).Render("●"),
					lipgloss.NewStyle().Foreground(gray).Render(key),
					lipgloss.NewStyle().Foreground(purple).Render(value)))
			}
		}

		// Combine columns side by side with proper spacing
		maxLines := len(leftColumn)
		if len(rightColumn) > maxLines {
			maxLines = len(rightColumn)
		}

		combinedLines := []string{}
		for i := 0; i < maxLines; i++ {
			left := ""
			right := ""
			if i < len(leftColumn) {
				left = leftColumn[i]
			}
			if i < len(rightColumn) {
				right = rightColumn[i]
			}

			// Create properly formatted line with consistent spacing
			if left != "" && right != "" {
				// Calculate actual display width (without ANSI codes)
				leftPlain := stripAnsi(left)
				leftPadding := 45 - len(leftPlain) // Target width for left column
				if leftPadding < 2 {
					leftPadding = 2 // Minimum spacing
				}
				combinedLines = append(combinedLines, "  "+left+strings.Repeat(" ", leftPadding)+right)
			} else if left != "" {
				combinedLines = append(combinedLines, "  "+left)
			} else if right != "" {
				combinedLines = append(combinedLines, strings.Repeat(" ", 47)+right)
			}
		}

		configContent := strings.Join(combinedLines, "\n")
		configSection = configHeader + "\n" + configContent + "\n"
	}

	// Status
	status := "🟢 SCANNING"
	statusStyle := lipgloss.NewStyle().Foreground(green)
	if m.stats.IsPaused {
		status = "⏸ PAUSED"
		statusStyle = lipgloss.NewStyle().Foreground(yellow)
	}

	// Calculate stats
	elapsed := time.Since(m.stats.StartTime)
	progress := float64(0)
	if m.stats.TotalIPs > 0 {
		progress = float64(m.stats.ScannedIPs) / float64(m.stats.TotalIPs) * 100
	}

	successRate := float64(0)
	if m.stats.ScannedIPs > 0 {
		successRate = float64(m.stats.SuccessIPs) / float64(m.stats.ScannedIPs) * 100
	}

	ipsPerSecond := float64(0)
	if elapsed.Seconds() > 0 {
		ipsPerSecond = float64(m.stats.ScannedIPs) / elapsed.Seconds()
	}

	eta := time.Duration(0)
	if ipsPerSecond > 0 {
		remaining := m.stats.TotalIPs - m.stats.ScannedIPs
		eta = time.Duration(float64(remaining)/ipsPerSecond) * time.Second
	}

	// Build stats display
	progressBar := m.renderProgressBar(progress, 45)

	// Count active workers
	activeWorkers := 0
	for _, worker := range m.stats.Workers {
		if time.Since(worker.LastSeen) < 15*time.Second {
			activeWorkers++
		}
	}

	stats := fmt.Sprintf(`%s  %s

  Progress:     %d / %d  IPs  (%s)  │  Workers: %s
  Success Rate: %s  (Hits: %s  Fails: %s)
  Speed:        %s  IPs/sec  │  Elapsed: %s  │  ETA: %s
`,
		statusStyle.Render(status),
		progressBar,
		m.stats.ScannedIPs,
		m.stats.TotalIPs,
		lipgloss.NewStyle().Foreground(purple).Render(fmt.Sprintf("%.1f%%", progress)),
		lipgloss.NewStyle().Foreground(cyan).Bold(true).Render(fmt.Sprintf("%d/%d", activeWorkers, m.stats.TotalWorkers)),
		lipgloss.NewStyle().Foreground(green).Render(fmt.Sprintf("%.1f%%", successRate)),
		lipgloss.NewStyle().Foreground(green).Bold(true).Render(fmt.Sprintf("%d", m.stats.SuccessIPs)),
		lipgloss.NewStyle().Foreground(red).Bold(true).Render(fmt.Sprintf("%d", m.stats.FailedIPs)),
		lipgloss.NewStyle().Foreground(purple).Render(fmt.Sprintf("%.2f", ipsPerSecond)),
		lipgloss.NewStyle().Foreground(gray).Render(formatDuration(elapsed)),
		lipgloss.NewStyle().Foreground(gray).Render(formatDuration(eta)),
	)

	// Workers section
	workers := lipgloss.NewStyle().
		Foreground(cyan).
		Bold(true).
		MarginTop(1).
		Render("🔧 WORKERS")

	workerContent := ""
	for i := 0; i < m.stats.TotalWorkers; i++ {
		if worker, exists := m.stats.Workers[i]; exists {
			isActive := time.Since(worker.LastSeen) < 15*time.Second
			statusIcon := "⚡"
			statusColor := green
			currentIP := worker.CurrentIP

			if !isActive || currentIP == "" {
				statusIcon = "💤"
				statusColor = gray
				currentIP = "idle"
			}

			if m.stats.IsPaused {
				statusIcon = "⏸"
				statusColor = yellow
			}

			workerContent += fmt.Sprintf("  %s Worker %d: %s\n",
				lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon),
				i+1,
				lipgloss.NewStyle().Foreground(purple).Render(currentIP))
		}
	}

	// Activity section
	activity := lipgloss.NewStyle().
		Foreground(purple).
		Bold(true).
		MarginTop(1).
		Render("📋 RECENT ACTIVITY")

	logContent := ""
	startIdx := 0
	if len(m.logs) > m.maxLogs {
		startIdx = len(m.logs) - m.maxLogs
	}

	for i := startIdx; i < len(m.logs); i++ {
		log := m.logs[i]
		timeStr := log.Timestamp.Format("15:04:05")
		var styledMsg string

		switch log.Type {
		case LogSuccess:
			styledMsg = lipgloss.NewStyle().Foreground(green).Render(log.Message)
		case LogFail:
			styledMsg = lipgloss.NewStyle().Foreground(red).Render(log.Message)
		case LogError:
			styledMsg = lipgloss.NewStyle().Foreground(red).Render(log.Message)
		case LogStartup:
			styledMsg = lipgloss.NewStyle().Foreground(cyan).Render(log.Message)
		default:
			styledMsg = lipgloss.NewStyle().Foreground(gray).Render(log.Message)
		}

		if log.IP != "" {
			logContent += fmt.Sprintf("  %s  │  %s  │  %s\n",
				lipgloss.NewStyle().Foreground(gray).Render(timeStr),
				log.IP,
				styledMsg)
		}
	}

	controls := lipgloss.NewStyle().
		MarginTop(1).
		Foreground(gray).
		Render("Controls:  [P] Pause/Resume  │  [Q/Esc] Quit")

	return title + "\n" + configSection + stats + "\n" + workers + "\n" + workerContent + activity + "\n" + logContent + controls
}

// Helper functions
func (m *Model) addLog(entry LogEntry) {
	m.logs = append(m.logs, entry)
}

func (m *Model) togglePause() tea.Cmd {
	return func() tea.Msg {
		return PauseToggle{}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) renderProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	return fmt.Sprintf("[%s]", bar)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}

	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// UpdateStats safely updates stats from external goroutines
func (m *Model) UpdateStats(update ScanUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.ScannedIPs++
	// Update worker info
	if worker, exists := m.stats.Workers[update.WorkerID]; exists {
		worker.CurrentIP = update.IP
		worker.LastSeen = time.Now()
	}

	if update.Success {
		m.stats.SuccessIPs++
	} else {
		m.stats.FailedIPs++
	}
}

// GetContext returns the model's context for cancellation
func (m *Model) GetContext() context.Context {
	return m.ctx
}

// GetStopChannel returns the stop channel
func (m *Model) GetStopChannel() <-chan struct{} {
	return m.stopCh
}

// IsPaused returns current pause state
func (m *Model) IsPaused() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats.IsPaused
}

// GetStats returns a copy of current stats
func (m *Model) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// stripAnsi removes ANSI escape codes to calculate actual text width
func stripAnsi(s string) string {
	// Simple regex to remove ANSI codes - for width calculation only
	ansiRegex := strings.NewReplacer(
		"\033[0m", "", "\033[1m", "", "\033[22m", "",
		"\033[31m", "", "\033[32m", "", "\033[33m", "",
		"\033[34m", "", "\033[35m", "", "\033[36m", "",
		"\033[37m", "", "\033[90m", "", "\033[91m", "",
		"\033[92m", "", "\033[93m", "", "\033[94m", "",
		"\033[95m", "", "\033[96m", "", "\033[97m", "",
	)
	// Remove common lipgloss color codes
	result := ansiRegex.Replace(s)
	// Remove any remaining escape sequences (basic pattern)
	for strings.Contains(result, "\033[") {
		start := strings.Index(result, "\033[")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return result
}
