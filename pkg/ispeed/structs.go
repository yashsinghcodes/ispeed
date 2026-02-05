package ispeed

import "time"

const (
	DefaultServerAddr = ":8080"
	DefaultClientBase = "https://speed.getanswers.pro"
	DefaultDuration   = 12 * time.Second
	DefaultStreams    = 1
	DefaultChunkSize  = 64 * 1024
	DefaultDownloadMB = 40
	DefaultPingCount  = 6
	DefaultTimeout    = 30 * time.Second
	DefaultMaxBytes   = int64(1024 * 1024 * 1024)
	DefaultReadLimit  = int64(512 * 1024 * 1024)
)

type ServerConfig struct {
	Addr      string
	MaxBytes  int64
	ReadLimit int64
}

type ClientConfig struct {
	BaseURL    string
	Duration   time.Duration
	Streams    int
	ChunkSize  int
	DownloadMB int
	PingCount  int
	Timeout    time.Duration
	JSON       bool
	Progress   func(ProgressUpdate)
}

type ProgressUpdate struct {
	Phase   string
	Percent float64
	Mbps    float64
	PingMs  float64
}

type PingMetrics struct {
	Min time.Duration
	Avg time.Duration
	P95 time.Duration
}

type SpeedMetrics struct {
	Mbps     float64
	Bytes    int64
	Duration time.Duration
}

type Result struct {
	Ping     PingMetrics
	Download SpeedMetrics
	Upload   SpeedMetrics
}
