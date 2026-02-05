package ispeed

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func RunClient(cfg ClientConfig) (Result, error) {
	cfg = normalizeClientConfig(cfg)
	client := &http.Client{Timeout: cfg.Timeout}

	pingRes, err := runPing(client, cfg)
	if err != nil {
		return Result{}, err
	}

	downloadRes, err := runDownload(client, cfg)
	if err != nil {
		return Result{}, err
	}

	uploadRes, err := runUpload(client, cfg)
	if err != nil {
		return Result{}, err
	}

	return Result{Ping: pingRes, Download: downloadRes, Upload: uploadRes}, nil
}

func normalizeClientConfig(cfg ClientConfig) ClientConfig {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultClientBase
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Duration <= 0 {
		cfg.Duration = DefaultDuration
	}
	if cfg.Streams < 1 {
		cfg.Streams = DefaultStreams
	}
	if cfg.ChunkSize < 1024 {
		cfg.ChunkSize = DefaultChunkSize
	}
	if cfg.DownloadMB < 1 {
		cfg.DownloadMB = DefaultDownloadMB
	}
	if cfg.PingCount < 1 {
		cfg.PingCount = DefaultPingCount
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}

	return cfg
}

func runPing(client *http.Client, cfg ClientConfig) (PingMetrics, error) {
	results := make([]time.Duration, 0, cfg.PingCount)
	url := cfg.BaseURL + "/ping"

	for i := 0; i < cfg.PingCount; i++ {
		start := time.Now()
		resp, err := client.Get(url)
		if err != nil {
			return PingMetrics{}, err
		}

		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		results = append(results, time.Since(start))
		if i < cfg.PingCount-1 {
			time.Sleep(150 * time.Millisecond)
		}
	}

	// No assert :(
	if len(results) == 0 {
		return PingMetrics{}, errors.New("ping returned no data")
	}

	slices.Sort(results)
	min := results[0]
	avg := avgDuration(results)
	p95 := percentileDuration(results, 0.95)

	return PingMetrics{Min: min, Avg: avg, P95: p95}, nil
}

func setRunErr(errOnce *sync.Once, runErr *error, err error) {
	if err == nil {
		return
	}
	errOnce.Do(func() {
		*runErr = err
	})
}

func runDownload(client *http.Client, cfg ClientConfig) (SpeedMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration+5*time.Second)
	defer cancel()

	var totalBytes int64
	var runErr error
	var errOnce sync.Once
	wg := sync.WaitGroup{}
	start := time.Now()

	perStreamBytes := int64(cfg.DownloadMB) * 1024 * 1024

	for i := 0; i < cfg.Streams; i++ {
		wg.Go(func() {
			url := fmt.Sprintf("%s/download?size=%d", cfg.BaseURL, perStreamBytes)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				setRunErr(&errOnce, &runErr, err)
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				setRunErr(&errOnce, &runErr, err)
				return
			}

			buf := make([]byte, cfg.ChunkSize)
			for {
				read, err := resp.Body.Read(buf)
				if read > 0 {
					atomic.AddInt64(&totalBytes, int64(read))
				}
				if err != nil {
					if !errors.Is(err, io.EOF) {
						setRunErr(&errOnce, &runErr, err)
					}
					break
				}
			}
			_ = resp.Body.Close()
		})
	}

	wg.Wait()
	elapsed := time.Since(start)

	if runErr != nil {
		return SpeedMetrics{}, runErr
	}
	if totalBytes == 0 {
		return SpeedMetrics{}, errors.New("download returned no data")
	}

	mbps := bytesToMbps(totalBytes, elapsed)

	return SpeedMetrics{Mbps: mbps, Bytes: totalBytes, Duration: elapsed}, nil
}

func runUpload(client *http.Client, cfg ClientConfig) (SpeedMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration+5*time.Second)
	defer cancel()

	var totalBytes int64
	var runErr error
	var errOnce sync.Once
	wg := sync.WaitGroup{}
	start := time.Now()

	for i := 0; i < cfg.Streams; i++ {
		wg.Go(func() {
			uploadCtx, cancelUpload := context.WithTimeout(ctx, cfg.Duration)
			defer cancelUpload()

			reader := &timedReader{ctx: uploadCtx, chunkSize: cfg.ChunkSize}
			req, err := http.NewRequestWithContext(uploadCtx, http.MethodPost, cfg.BaseURL+"/upload", reader)
			if err != nil {
				setRunErr(&errOnce, &runErr, err)
				return
			}
			req.Header.Set("Content-Type", "application/octet-stream")
			resp, err := client.Do(req)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
					atomic.AddInt64(&totalBytes, reader.bytes())
					return
				}
				setRunErr(&errOnce, &runErr, err)
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			atomic.AddInt64(&totalBytes, reader.bytes())
		})
	}

	wg.Wait()
	elapsed := time.Since(start)

	if runErr != nil {
		return SpeedMetrics{}, runErr
	}
	if totalBytes == 0 {
		return SpeedMetrics{}, errors.New("upload sent no data")
	}

	mbps := bytesToMbps(totalBytes, elapsed)

	return SpeedMetrics{Mbps: mbps, Bytes: totalBytes, Duration: elapsed}, nil
}

func avgDuration(items []time.Duration) time.Duration {
	if len(items) == 0 {
		return 0
	}
	var total time.Duration
	for _, item := range items {
		total += item
	}
	return time.Duration(int64(total) / int64(len(items)))
}

func percentileDuration(items []time.Duration, percentile float64) time.Duration {
	if len(items) == 0 {
		return 0
	}
	if percentile <= 0 {
		return items[0]
	}
	if percentile >= 1 {
		return items[len(items)-1]
	}
	index := int(math.Ceil(float64(len(items))*percentile)) - 1
	index = max(index, 0)

	if index >= len(items) {
		index = len(items) - 1
	}
	return items[index]
}

func bytesToMbps(bytes int64, duration time.Duration) float64 {
	if duration <= 0 {
		return 0
	}
	bits := float64(bytes) * 8
	return bits / duration.Seconds() / 1_000_000
}

type timedReader struct {
	ctx       context.Context
	chunkSize int
	count     int64
}

func (t *timedReader) Read(p []byte) (int, error) {
	if t.ctx.Err() != nil {
		return 0, t.ctx.Err()
	}

	if len(p) > t.chunkSize {
		p = p[:t.chunkSize]
	}

	_, err := rand.Read(p)
	if err != nil {
		return 0, err
	}
	atomic.AddInt64(&t.count, int64(len(p)))
	return len(p), nil
}

func (t *timedReader) bytes() int64 {
	return atomic.LoadInt64(&t.count)
}
