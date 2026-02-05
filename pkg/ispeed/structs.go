package ispeed

import "time"

const (
	DefaultServerAddr = ":8080"
	DefaultClientBase = "http://localhost:8080"
	DefaultDuration   = 12 * time.Second
	DefaultStreams    = 4
	DefaultChunkSize  = 64 * 1024
	DefaultDownloadMB = 50
	DefaultPingCount  = 6
	DefaultTimeout    = 5 * time.Second
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
