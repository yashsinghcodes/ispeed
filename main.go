package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yashsinghcodes/ispeed/pkg/ispeed"
	"gopkg.in/yaml.v3"
)

type progressMsg struct {
	update ispeed.ProgressUpdate
}

type resultMsg struct {
	result ispeed.Result
}

type errMsg struct {
	err error
}

type progressState struct {
	percent float64
	mbps    float64
}

type serverList struct {
	Servers []serverEntry `yaml:"servers"`
}

type serverEntry struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type model struct {
	cfg          ispeed.ClientConfig
	progressCh   <-chan ispeed.ProgressUpdate
	progressDone <-chan struct{}
	width        int
	ping         progressState
	download     progressState
	upload       progressState
	result       *ispeed.Result
	err          error
}

func newModel(cfg ispeed.ClientConfig, progressCh <-chan ispeed.ProgressUpdate, progressDone <-chan struct{}) model {
	return model{
		cfg:          cfg,
		progressCh:   progressCh,
		progressDone: progressDone,
		width:        72,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(listenProgress(m.progressCh), listenDone(m.progressDone))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		return m, nil
	case progressMsg:
		switch typed.update.Phase {
		case "ping":
			m.ping.percent = typed.update.Percent
			m.ping.mbps = typed.update.PingMs
		case "download":
			m.download.percent = typed.update.Percent
			m.download.mbps = typed.update.Mbps
		case "upload":
			m.upload.percent = typed.update.Percent
			m.upload.mbps = typed.update.Mbps
		}
		return m, listenProgress(m.progressCh)
	case resultMsg:
		if typed.result.Ping.Min != 0 || typed.result.Download.Mbps != 0 || typed.result.Upload.Mbps != 0 {
			m.result = &typed.result
		}
		return m, tea.Quit
	case errMsg:
		m.err = typed.err
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")).Render("ispeed")
	subtitle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(m.cfg.BaseURL)

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		return fmt.Sprintf("%s\n%s\n\n%s\n", title, subtitle, errorStyle.Render(m.err.Error()))
	}

	content := []string{title, subtitle, ""}
	content = append(content, renderPingLine(m.ping.percent, m.cfg.PingCount, m.ping.mbps))
	content = append(content, renderSpeedLine("Download", m.download.mbps))
	content = append(content, renderSpeedLine("Upload", m.upload.mbps))

	return strings.Join(content, "\n") + "\n"
}

func listenProgress(ch <-chan ispeed.ProgressUpdate) tea.Cmd {
	return func() tea.Msg {
		update, ok := <-ch
		if !ok {
			return nil
		}
		return progressMsg{update: update}
	}
}

func listenDone(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-ch
		return resultMsg{}
	}
}

func renderPingLine(percent float64, total int, pingMs float64) string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	current := int(math.Round((percent / 100) * float64(total)))
	if current < 0 {
		current = 0
	}

	if current > total {
		current = total
	}
	progressText := valueStyle.Render(fmt.Sprintf("%d/%d", current, total))
	pingText := accentStyle.Render(fmt.Sprintf("%6.2f ms", pingMs))
	return fmt.Sprintf("%s %s  %s", labelStyle.Render("Ping"), progressText, pingText)
}

func renderSpeedLine(label string, mbps float64) string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	return fmt.Sprintf("%-8s %s", labelStyle.Render(label), valueStyle.Render(fmt.Sprintf("%6.2f Mbps", mbps)))
}

func configPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".ispeed.yaml"), nil
}

func loadServerList() (serverList, error) {
	path, err := configPath()
	if err != nil {
		return serverList{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[ERROR] Failed to read config file at ~/.ispeed.yaml")
		data = []byte(defaultConfig())
	}

	var list serverList
	if err := yaml.Unmarshal(data, &list); err != nil {
		return serverList{}, err
	}

	return list, nil
}

func defaultConfig() (string) {
	return "servers:\n  - name: Default\n    url: https://speed.getanswers.pro\n"
}

func pickFastestServer() (string, error) {
	list, err := loadServerList()
	if err != nil {
		return "", fmt.Errorf("read server list: %w", err)
	}

	if len(list.Servers) == 0 {
		return "", fmt.Errorf("no servers defined in config")
	}

	client := &http.Client{Timeout: 4 * time.Second}
	bestURL := ""
	bestLatency := time.Duration(1<<63 - 1)

	for _, server := range list.Servers {
		if server.URL == "" {
			continue
		}
		start := time.Now()
		resp, err := client.Get(strings.TrimRight(server.URL, "/") + "/ping")
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		elapsed := time.Since(start)
		if elapsed < bestLatency {
			bestLatency = elapsed
			bestURL = strings.TrimRight(server.URL, "/")
		}
	}

	if bestURL == "" {
		return "", fmt.Errorf("no reachable servers found")
	}

	return bestURL, nil
}


func main() {
	f, err := os.OpenFile("/tmp/ispeed.log", os.O_CREATE | os.O_RDWR, os.ModeTemporary)
	if err != nil {
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg := parseFlags()

	if cfg.BaseURL == "" {
		selected, err := pickFastestServer()
		if err != nil {
			log.Fatalf("[ERROR] failed to select server: %v", err)
		}
		cfg.BaseURL = selected
	}

	if cfg.JSON {
		result, err := ispeed.RunClient(cfg)
		if err != nil {
			log.Fatalf("[ERROR] speed test failed: %v", err)
		}
		fmt.Printf("{\"ping_ms\":%.2f,\"ping_avg_ms\":%.2f,\"ping_p95_ms\":%.2f,\"download_mbps\":%.2f,\"upload_mbps\":%.2f}\n",
			float64(result.Ping.Min.Milliseconds()), float64(result.Ping.Avg.Milliseconds()), float64(result.Ping.P95.Milliseconds()), result.Download.Mbps, result.Upload.Mbps)
		return
	}

	progressCh := make(chan ispeed.ProgressUpdate, 16)
	progressDone := make(chan struct{})
	sendProgress := func(update ispeed.ProgressUpdate) {
		select {
		case progressCh <- update:
		default:
		}
	}
	cfg.Progress = func(update ispeed.ProgressUpdate) {
		sendProgress(update)
	}

	m := newModel(cfg, progressCh, progressDone)
	program := tea.NewProgram(m)

	go func() {
		result, err := ispeed.RunClient(cfg)
		if err != nil {
			program.Send(errMsg{err: err})
			close(progressDone)
			return
		}
		program.Send(resultMsg{result: result})
		close(progressDone)
	}()

	finalModel, err := program.Run()
	if err != nil {
		log.Fatalf("[ERROR] ui failed: %v", err)
	}
	close(progressCh)
	fmt.Print("\r\033[2K\n")
	if finished, ok := finalModel.(model); ok {
		if finished.err != nil {
			fmt.Fprintln(os.Stderr, finished.err.Error())
			os.Exit(1)
		}
	}
}

func parseFlags() ispeed.ClientConfig {
	baseURL := flag.String("url", "", "base URL for server (leave empty for auto-select)")
	duration := flag.Duration("duration", ispeed.DefaultDuration, "test duration")
	streams := flag.Int("streams", ispeed.DefaultStreams, "parallel streams")
	chunkSize := flag.Int("chunk-size", ispeed.DefaultChunkSize, "chunk size in bytes")
	downloadMB := flag.Int("download-mb", ispeed.DefaultDownloadMB, "download size per stream in MB")
	pingCount := flag.Int("ping-count", ispeed.DefaultPingCount, "number of ping samples")
	timeout := flag.Duration("timeout", ispeed.DefaultTimeout, "request timeout")
	jsonOut := flag.Bool("json", false, "print JSON output")
	flag.Parse()

	return ispeed.ClientConfig{
		BaseURL:    strings.TrimRight(*baseURL, "/"),
		Duration:   *duration,
		Streams:    *streams,
		ChunkSize:  *chunkSize,
		DownloadMB: *downloadMB,
		PingCount:  *pingCount,
		Timeout:    *timeout,
		JSON:       *jsonOut,
	}
}
