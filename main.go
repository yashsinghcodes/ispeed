package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/yashsinghcodes/ispeed/pkg/ispeed"
)

func main() {
	cfg := parseFlags()
	result, err := ispeed.RunClient(cfg)
	if err != nil {
		log.Fatalf("[ERROR] speed test failed: %v", err)
	}

	if cfg.JSON {
		fmt.Printf("{\"ping_ms\":%.2f,\"ping_avg_ms\":%.2f,\"ping_p95_ms\":%.2f,\"download_mbps\":%.2f,\"upload_mbps\":%.2f}\n",
			float64(result.Ping.Min.Milliseconds()), float64(result.Ping.Avg.Milliseconds()), float64(result.Ping.P95.Milliseconds()), result.Download.Mbps, result.Upload.Mbps)
		return
	}

	fmt.Println("Results")
	fmt.Printf("  Ping:    min %v | avg %v | p95 %v\n", result.Ping.Min, result.Ping.Avg, result.Ping.P95)
	fmt.Printf("  Download: %.2f Mbps\n", result.Download.Mbps)
	fmt.Printf("  Upload:   %.2f Mbps\n", result.Upload.Mbps)
}

func parseFlags() ispeed.ClientConfig {
	baseURL := flag.String("url", ispeed.DefaultClientBase, "base URL for server")
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
