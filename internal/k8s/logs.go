package k8s

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

type LogLine struct {
	Timestamp time.Time
	Container string
	Content   string
	IsError   bool
}

type LogOptions struct {
	Container  string
	TailLines  int64
	Since      time.Duration
	Previous   bool
	Follow     bool
	Timestamps bool
}

func DefaultLogOptions() LogOptions {
	return LogOptions{
		TailLines:  100,
		Timestamps: true,
	}
}

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

func parseLogStream(reader io.Reader, container string, hasTimestamps bool) ([]LogLine, error) {
	var lines []LogLine
	scanner := bufio.NewScanner(reader)

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		logLine := LogLine{
			Container: container,
			Content:   line,
		}

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
			continue
		}
		allLogs = append(allLogs, logs...)
	}

	sortLogsByTime(allLogs)
	return allLogs, nil
}

func sortLogsByTime(logs []LogLine) {
	for i := 0; i < len(logs)-1; i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[j].Timestamp.Before(logs[i].Timestamp) {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}
}

func GetPreviousLogs(ctx context.Context, clientset *kubernetes.Clientset, namespace, podName, container string, tailLines int64) ([]LogLine, error) {
	opts := LogOptions{
		Container:  container,
		TailLines:  tailLines,
		Previous:   true,
		Timestamps: true,
	}
	return GetPodLogs(ctx, clientset, namespace, podName, opts)
}

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

func FilterErrorLogs(logs []LogLine) []LogLine {
	var errors []LogLine
	for _, log := range logs {
		if log.IsError {
			errors = append(errors, log)
		}
	}
	return errors
}

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
