package repository

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// LogLine represents a single line from container logs.
// It includes parsed metadata such as timestamp and container name,
// plus a flag indicating if the line appears to contain an error.
type LogLine struct {
	Timestamp time.Time // Parsed timestamp from the log line
	Container string    // Name of the container that produced this log
	Content   string    // The actual log message content
	IsError   bool      // True if the line contains error-related keywords
}

// LogOptions configures how container logs are retrieved.
type LogOptions struct {
	Container  string        // Specific container name (empty for default)
	TailLines  int64         // Number of lines to fetch from the end
	Since      time.Duration // Only return logs newer than this duration
	Previous   bool          // Fetch logs from the previous container instance
	Follow     bool          // Stream logs in real-time (not implemented in batch mode)
	Timestamps bool          // Include timestamps in log output
}

// DefaultLogOptions returns a LogOptions with sensible defaults:
// 100 tail lines with timestamps enabled.
func DefaultLogOptions() LogOptions {
	return LogOptions{
		TailLines:  100,
		Timestamps: true,
	}
}

// GetPodLogs retrieves container logs for a specific pod.
// It returns parsed log lines with timestamps and error detection.
func GetPodLogs(ctx context.Context, clientset *kubernetes.Clientset, namespace, podName string, opts LogOptions) ([]LogLine, error) {
	podLogOpts := &corev1.PodLogOptions{
		Container:  opts.Container,
		Previous:   opts.Previous,
		Timestamps: opts.Timestamps,
	}

	if opts.TailLines > 0 {
		podLogOpts.TailLines = &opts.TailLines
	}

	if opts.Since > 0 {
		sinceSeconds := int64(opts.Since.Seconds())
		podLogOpts.SinceSeconds = &sinceSeconds
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, podLogOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	defer stream.Close()

	return parseLogStream(stream, opts.Container, opts.Timestamps)
}

// parseLogStream reads log lines from a stream and parses them into LogLine structs.
// It handles timestamp parsing in both RFC3339Nano and RFC3339 formats.
func parseLogStream(reader io.Reader, container string, hasTimestamps bool) ([]LogLine, error) {
	var lines []LogLine
	scanner := bufio.NewScanner(reader)

	// Increase buffer size to handle long log lines (up to 1MB)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		logLine := LogLine{
			Container: container,
			Content:   line,
		}

		// Parse timestamp if present (format: 2006-01-02T15:04:05.999999999Z)
		if hasTimestamps && len(line) > 30 {
			if ts, err := time.Parse(time.RFC3339Nano, line[:30]); err == nil {
				logLine.Timestamp = ts
				logLine.Content = strings.TrimSpace(line[31:])
			} else if ts, err := time.Parse(time.RFC3339, line[:20]); err == nil {
				logLine.Timestamp = ts
				logLine.Content = strings.TrimSpace(line[21:])
			}
		}

		logLine.IsError = isErrorLine(logLine.Content)
		lines = append(lines, logLine)
	}

	return lines, scanner.Err()
}

// isErrorLine checks if a log line contains common error indicators.
// It performs case-insensitive matching against keywords like "error", "fatal", "panic", etc.
func isErrorLine(content string) bool {
	lower := strings.ToLower(content)
	errorIndicators := []string{
		"error", "err:", "fatal", "panic", "exception",
		"failed", "failure", "crash", "critical",
	}
	for _, indicator := range errorIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

// GetAllContainerLogs retrieves logs from all containers in a pod.
// It distributes the tail line limit evenly across containers and merges
// the results sorted by timestamp.
func GetAllContainerLogs(ctx context.Context, clientset *kubernetes.Clientset, namespace, podName string, tailLines int64) ([]LogLine, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var allLogs []LogLine
	linesPerContainer := tailLines / int64(len(pod.Spec.Containers))
	if linesPerContainer < 10 {
		linesPerContainer = 10
	}

	for _, container := range pod.Spec.Containers {
		opts := LogOptions{
			Container:  container.Name,
			TailLines:  linesPerContainer,
			Timestamps: true,
		}

		logs, err := GetPodLogs(ctx, clientset, namespace, podName, opts)
		if err != nil {
			continue // Skip containers that fail (e.g., not started yet)
		}
		allLogs = append(allLogs, logs...)
	}

	sortLogsByTime(allLogs)
	return allLogs, nil
}

// sortLogsByTime sorts log lines chronologically by their timestamp.
// Uses simple bubble sort which is adequate for typical log volumes.
func sortLogsByTime(logs []LogLine) {
	for i := 0; i < len(logs)-1; i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[j].Timestamp.Before(logs[i].Timestamp) {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}
}

// GetPreviousLogs retrieves logs from the previous instance of a container.
// This is useful for debugging crashes where the current container has no logs.
func GetPreviousLogs(ctx context.Context, clientset *kubernetes.Clientset, namespace, podName, container string, tailLines int64) ([]LogLine, error) {
	opts := LogOptions{
		Container:  container,
		TailLines:  tailLines,
		Previous:   true,
		Timestamps: true,
	}
	return GetPodLogs(ctx, clientset, namespace, podName, opts)
}

// SearchLogs filters log lines that contain the given query string.
// The search is case-insensitive. Returns all logs if query is empty.
func SearchLogs(logs []LogLine, query string) []LogLine {
	if query == "" {
		return logs
	}

	query = strings.ToLower(query)
	var matches []LogLine
	for _, log := range logs {
		if strings.Contains(strings.ToLower(log.Content), query) {
			matches = append(matches, log)
		}
	}
	return matches
}

// FilterErrorLogs returns only log lines that have been flagged as errors.
func FilterErrorLogs(logs []LogLine) []LogLine {
	var errors []LogLine
	for _, log := range logs {
		if log.IsError {
			errors = append(errors, log)
		}
	}
	return errors
}

// GetLogsAroundTime returns log lines within a time window around the target time.
// Useful for investigating what happened before and after a specific event.
func GetLogsAroundTime(logs []LogLine, target time.Time, windowMinutes int) []LogLine {
	window := time.Duration(windowMinutes) * time.Minute
	start := target.Add(-window)
	end := target.Add(window)

	var result []LogLine
	for _, log := range logs {
		if !log.Timestamp.IsZero() && log.Timestamp.After(start) && log.Timestamp.Before(end) {
			result = append(result, log)
		}
	}
	return result
}
