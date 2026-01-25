package torrentium_client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/1amkhush/torrentium/pkg/logger"
)

// DownloadPriority represents the priority level of a download
type DownloadPriority int

const (
	PriorityLow    DownloadPriority = 0
	PriorityNormal DownloadPriority = 1
	PriorityHigh   DownloadPriority = 2
)

// DownloadStatus represents the status of a download
type DownloadStatus string

const (
	StatusQueued      DownloadStatus = "queued"
	StatusStarting    DownloadStatus = "starting"
	StatusDownloading DownloadStatus = "downloading"
	StatusPaused      DownloadStatus = "paused"
	StatusCompleted   DownloadStatus = "completed"
	StatusFailed      DownloadStatus = "failed"
	StatusCancelled   DownloadStatus = "cancelled"
)

// QueuedDownload represents a download in the queue
type QueuedDownload struct {
	CID             string
	Priority        DownloadPriority
	Status          DownloadStatus
	Progress        float64 // 0.0 to 1.0
	BytesDownloaded int64
	TotalBytes      int64
	PiecesCompleted int
	TotalPieces     int
	Speed           float64 // bytes per second
	ETA             time.Duration
	AddedAt         time.Time
	StartedAt       time.Time
	CompletedAt     time.Time
	Error           string
	MaxBandwidth    int64 // Per-download bandwidth limit (0 = no limit)
	SelectedFiles   []int // For multi-file downloads (-1 = all)
	mu              sync.RWMutex
	cancelFunc      context.CancelFunc
	pauseCh         chan struct{}
	resumeCh        chan struct{}
	speedSamples    []float64 // For calculating average speed
	lastSpeedUpdate time.Time
	lastBytesCount  int64
}

// DownloadQueue manages a priority queue of downloads
type DownloadQueue struct {
	downloads          []*QueuedDownload
	mu                 sync.RWMutex
	maxConcurrent      int
	activeCount        int
	onStatusChange     func(cid string, status DownloadStatus)
	onProgressUpdate   func(cid string, progress float64, speed float64)
	onComplete         func(cid string)
	onError            func(cid string, err error)
	globalMaxBandwidth int64
}

// NewDownloadQueue creates a new download queue
func NewDownloadQueue(maxConcurrent int) *DownloadQueue {
	if maxConcurrent <= 0 {
		maxConcurrent = 3
	}
	return &DownloadQueue{
		downloads:     make([]*QueuedDownload, 0),
		maxConcurrent: maxConcurrent,
	}
}

// SetCallbacks sets the callback functions for queue events
func (q *DownloadQueue) SetCallbacks(
	onStatusChange func(cid string, status DownloadStatus),
	onProgressUpdate func(cid string, progress float64, speed float64),
	onComplete func(cid string),
	onError func(cid string, err error),
) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.onStatusChange = onStatusChange
	q.onProgressUpdate = onProgressUpdate
	q.onComplete = onComplete
	q.onError = onError
}

// SetMaxConcurrent sets the maximum number of concurrent downloads
func (q *DownloadQueue) SetMaxConcurrent(max int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.maxConcurrent = max
	logger.Info().Int("max_concurrent", max).Msg("Updated max concurrent downloads")
}

// GetMaxConcurrent returns the maximum number of concurrent downloads
func (q *DownloadQueue) GetMaxConcurrent() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.maxConcurrent
}

// SetGlobalBandwidth sets the global bandwidth limit for all downloads
func (q *DownloadQueue) SetGlobalBandwidth(bytesPerSecond int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.globalMaxBandwidth = bytesPerSecond
}

// Add adds a new download to the queue
func (q *DownloadQueue) Add(cid string, priority DownloadPriority, maxBandwidth int64, selectedFiles []int) (*QueuedDownload, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if already in queue
	for _, dl := range q.downloads {
		if dl.CID == cid && dl.Status != StatusCompleted && dl.Status != StatusCancelled && dl.Status != StatusFailed {
			return nil, fmt.Errorf("download already in queue: %s", cid)
		}
	}

	download := &QueuedDownload{
		CID:           cid,
		Priority:      priority,
		Status:        StatusQueued,
		AddedAt:       time.Now(),
		MaxBandwidth:  maxBandwidth,
		SelectedFiles: selectedFiles,
		pauseCh:       make(chan struct{}),
		resumeCh:      make(chan struct{}),
		speedSamples:  make([]float64, 0, 10),
	}

	// Insert in priority order
	inserted := false
	for i, dl := range q.downloads {
		if priority > dl.Priority {
			q.downloads = append(q.downloads[:i], append([]*QueuedDownload{download}, q.downloads[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.downloads = append(q.downloads, download)
	}

	logger.Info().Str("cid", cid).Int("priority", int(priority)).Msg("Added download to queue")
	return download, nil
}

// Remove removes a download from the queue
func (q *DownloadQueue) Remove(cid string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, dl := range q.downloads {
		if dl.CID == cid {
			// Cancel if active
			dl.mu.Lock()
			if dl.cancelFunc != nil {
				dl.cancelFunc()
			}
			dl.Status = StatusCancelled
			dl.mu.Unlock()

			// Remove from queue
			q.downloads = append(q.downloads[:i], q.downloads[i+1:]...)
			logger.Info().Str("cid", cid).Msg("Removed download from queue")
			return nil
		}
	}
	return fmt.Errorf("download not found: %s", cid)
}

// Pause pauses a download
func (q *DownloadQueue) Pause(cid string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, dl := range q.downloads {
		if dl.CID == cid {
			dl.mu.Lock()
			if dl.Status == StatusDownloading {
				dl.Status = StatusPaused
				close(dl.pauseCh)
				dl.pauseCh = make(chan struct{}) // Reset for future use
				dl.mu.Unlock()

				q.activeCount--
				if q.onStatusChange != nil {
					go q.onStatusChange(cid, StatusPaused)
				}
				logger.Info().Str("cid", cid).Msg("Paused download")
				return nil
			}
			dl.mu.Unlock()
			return fmt.Errorf("download is not active: %s", cid)
		}
	}
	return fmt.Errorf("download not found: %s", cid)
}

// Resume resumes a paused download
func (q *DownloadQueue) Resume(cid string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, dl := range q.downloads {
		if dl.CID == cid {
			dl.mu.Lock()
			if dl.Status == StatusPaused {
				dl.Status = StatusQueued // Will be picked up by processor
				close(dl.resumeCh)
				dl.resumeCh = make(chan struct{}) // Reset for future use
				dl.mu.Unlock()

				if q.onStatusChange != nil {
					go q.onStatusChange(cid, StatusQueued)
				}
				logger.Info().Str("cid", cid).Msg("Resumed download")
				return nil
			}
			dl.mu.Unlock()
			return fmt.Errorf("download is not paused: %s", cid)
		}
	}
	return fmt.Errorf("download not found: %s", cid)
}

// SetPriority changes the priority of a download
func (q *DownloadQueue) SetPriority(cid string, priority DownloadPriority) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	var download *QueuedDownload
	var idx int

	for i, dl := range q.downloads {
		if dl.CID == cid {
			download = dl
			idx = i
			break
		}
	}

	if download == nil {
		return fmt.Errorf("download not found: %s", cid)
	}

	download.mu.Lock()
	download.Priority = priority
	download.mu.Unlock()

	// Re-sort the queue
	q.downloads = append(q.downloads[:idx], q.downloads[idx+1:]...)

	inserted := false
	for i, dl := range q.downloads {
		if priority > dl.Priority {
			q.downloads = append(q.downloads[:i], append([]*QueuedDownload{download}, q.downloads[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.downloads = append(q.downloads, download)
	}

	logger.Info().Str("cid", cid).Int("priority", int(priority)).Msg("Updated download priority")
	return nil
}

// SetBandwidth sets the bandwidth limit for a specific download
func (q *DownloadQueue) SetBandwidth(cid string, bytesPerSecond int64) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, dl := range q.downloads {
		if dl.CID == cid {
			dl.mu.Lock()
			dl.MaxBandwidth = bytesPerSecond
			dl.mu.Unlock()
			logger.Info().Str("cid", cid).Int64("bandwidth", bytesPerSecond).Msg("Updated download bandwidth limit")
			return nil
		}
	}
	return fmt.Errorf("download not found: %s", cid)
}

// GetNext returns the next download to process
func (q *DownloadQueue) GetNext() *QueuedDownload {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.activeCount >= q.maxConcurrent {
		return nil
	}

	for _, dl := range q.downloads {
		dl.mu.RLock()
		status := dl.Status
		dl.mu.RUnlock()

		if status == StatusQueued {
			dl.mu.Lock()
			dl.Status = StatusStarting
			dl.StartedAt = time.Now()
			dl.mu.Unlock()

			q.activeCount++
			return dl
		}
	}
	return nil
}

// UpdateProgress updates the progress of a download
func (q *DownloadQueue) UpdateProgress(cid string, bytesDownloaded, totalBytes int64, piecesCompleted, totalPieces int) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, dl := range q.downloads {
		if dl.CID == cid {
			dl.mu.Lock()

			// Calculate speed
			now := time.Now()
			if !dl.lastSpeedUpdate.IsZero() {
				elapsed := now.Sub(dl.lastSpeedUpdate).Seconds()
				if elapsed > 0 {
					bytesDelta := bytesDownloaded - dl.lastBytesCount
					speed := float64(bytesDelta) / elapsed

					// Keep rolling average of speed samples
					dl.speedSamples = append(dl.speedSamples, speed)
					if len(dl.speedSamples) > 10 {
						dl.speedSamples = dl.speedSamples[1:]
					}

					// Calculate average speed
					var totalSpeed float64
					for _, s := range dl.speedSamples {
						totalSpeed += s
					}
					dl.Speed = totalSpeed / float64(len(dl.speedSamples))
				}
			}

			dl.lastSpeedUpdate = now
			dl.lastBytesCount = bytesDownloaded
			dl.BytesDownloaded = bytesDownloaded
			dl.TotalBytes = totalBytes
			dl.PiecesCompleted = piecesCompleted
			dl.TotalPieces = totalPieces

			if totalBytes > 0 {
				dl.Progress = float64(bytesDownloaded) / float64(totalBytes)
			}

			// Calculate ETA
			if dl.Speed > 0 && totalBytes > 0 {
				remaining := totalBytes - bytesDownloaded
				dl.ETA = time.Duration(float64(remaining)/dl.Speed) * time.Second
			}

			speed := dl.Speed
			progress := dl.Progress
			dl.mu.Unlock()

			if q.onProgressUpdate != nil {
				go q.onProgressUpdate(cid, progress, speed)
			}
			return
		}
	}
}

// MarkCompleted marks a download as completed
func (q *DownloadQueue) MarkCompleted(cid string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, dl := range q.downloads {
		if dl.CID == cid {
			dl.mu.Lock()
			dl.Status = StatusCompleted
			dl.Progress = 1.0
			dl.CompletedAt = time.Now()
			dl.mu.Unlock()

			q.activeCount--

			if q.onStatusChange != nil {
				go q.onStatusChange(cid, StatusCompleted)
			}
			if q.onComplete != nil {
				go q.onComplete(cid)
			}
			logger.Info().Str("cid", cid).Msg("Download completed")
			return
		}
	}
}

// MarkFailed marks a download as failed
func (q *DownloadQueue) MarkFailed(cid string, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, dl := range q.downloads {
		if dl.CID == cid {
			dl.mu.Lock()
			dl.Status = StatusFailed
			dl.Error = err.Error()
			dl.mu.Unlock()

			q.activeCount--

			if q.onStatusChange != nil {
				go q.onStatusChange(cid, StatusFailed)
			}
			if q.onError != nil {
				go q.onError(cid, err)
			}
			logger.Error().Str("cid", cid).Err(err).Msg("Download failed")
			return
		}
	}
}

// MarkDownloading marks a download as actively downloading
func (q *DownloadQueue) MarkDownloading(cid string) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, dl := range q.downloads {
		if dl.CID == cid {
			dl.mu.Lock()
			dl.Status = StatusDownloading
			dl.mu.Unlock()

			if q.onStatusChange != nil {
				go q.onStatusChange(cid, StatusDownloading)
			}
			return
		}
	}
}

// Get returns a download by CID
func (q *DownloadQueue) Get(cid string) *QueuedDownload {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, dl := range q.downloads {
		if dl.CID == cid {
			return dl
		}
	}
	return nil
}

// GetAll returns all downloads in the queue
func (q *DownloadQueue) GetAll() []*QueuedDownload {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]*QueuedDownload, len(q.downloads))
	copy(result, q.downloads)
	return result
}

// GetActive returns all active downloads
func (q *DownloadQueue) GetActive() []*QueuedDownload {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var active []*QueuedDownload
	for _, dl := range q.downloads {
		dl.mu.RLock()
		if dl.Status == StatusDownloading || dl.Status == StatusStarting {
			active = append(active, dl)
		}
		dl.mu.RUnlock()
	}
	return active
}

// GetQueued returns all queued downloads
func (q *DownloadQueue) GetQueued() []*QueuedDownload {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var queued []*QueuedDownload
	for _, dl := range q.downloads {
		dl.mu.RLock()
		if dl.Status == StatusQueued {
			queued = append(queued, dl)
		}
		dl.mu.RUnlock()
	}
	return queued
}

// GetPaused returns all paused downloads
func (q *DownloadQueue) GetPaused() []*QueuedDownload {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var paused []*QueuedDownload
	for _, dl := range q.downloads {
		dl.mu.RLock()
		if dl.Status == StatusPaused {
			paused = append(paused, dl)
		}
		dl.mu.RUnlock()
	}
	return paused
}

// ActiveCount returns the number of active downloads
func (q *DownloadQueue) ActiveCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.activeCount
}

// QueuedCount returns the number of queued downloads
func (q *DownloadQueue) QueuedCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	count := 0
	for _, dl := range q.downloads {
		dl.mu.RLock()
		if dl.Status == StatusQueued {
			count++
		}
		dl.mu.RUnlock()
	}
	return count
}

// Clear removes all completed, failed, and cancelled downloads
func (q *DownloadQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	var remaining []*QueuedDownload
	for _, dl := range q.downloads {
		dl.mu.RLock()
		if dl.Status != StatusCompleted && dl.Status != StatusFailed && dl.Status != StatusCancelled {
			remaining = append(remaining, dl)
		}
		dl.mu.RUnlock()
	}
	q.downloads = remaining
	logger.Info().Int("remaining", len(remaining)).Msg("Cleared completed/failed downloads from queue")
}

// QueueInfo provides a snapshot of queue information
type QueueInfo struct {
	CID             string
	Status          DownloadStatus
	Priority        DownloadPriority
	Progress        float64
	BytesDownloaded int64
	TotalBytes      int64
	PiecesCompleted int
	TotalPieces     int
	Speed           float64
	ETA             time.Duration
	MaxBandwidth    int64
	Error           string
	AddedAt         time.Time
	StartedAt       time.Time
	CompletedAt     time.Time
}

// GetInfo returns a snapshot of download information
func (dl *QueuedDownload) GetInfo() QueueInfo {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	return QueueInfo{
		CID:             dl.CID,
		Status:          dl.Status,
		Priority:        dl.Priority,
		Progress:        dl.Progress,
		BytesDownloaded: dl.BytesDownloaded,
		TotalBytes:      dl.TotalBytes,
		PiecesCompleted: dl.PiecesCompleted,
		TotalPieces:     dl.TotalPieces,
		Speed:           dl.Speed,
		ETA:             dl.ETA,
		MaxBandwidth:    dl.MaxBandwidth,
		Error:           dl.Error,
		AddedAt:         dl.AddedAt,
		StartedAt:       dl.StartedAt,
		CompletedAt:     dl.CompletedAt,
	}
}

// IsPaused returns whether the download is paused
func (dl *QueuedDownload) IsPaused() bool {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	return dl.Status == StatusPaused
}

// SetCancelFunc sets the cancel function for the download
func (dl *QueuedDownload) SetCancelFunc(cancel context.CancelFunc) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	dl.cancelFunc = cancel
}

// WaitForPause blocks until the download is paused or resumed
func (dl *QueuedDownload) WaitForPause() {
	<-dl.pauseCh
}

// WaitForResume blocks until the download is resumed
func (dl *QueuedDownload) WaitForResume() {
	<-dl.resumeCh
}
