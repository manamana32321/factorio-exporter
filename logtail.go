package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"regexp"
	"time"
)

var (
	chatPattern     = regexp.MustCompile(`\[CHAT\]\s+(.+?):\s+(.+)`)
	joinPattern     = regexp.MustCompile(`(.+?)\s+joined the game`)
	leavePattern    = regexp.MustCompile(`(.+?)\s+left the game`)
	researchPattern = regexp.MustCompile(`Research finished:\s+(.+)`)
	rocketPattern   = regexp.MustCompile(`Rocket launched`)
	savePattern     = regexp.MustCompile(`Saving game as\s+(.+)`)
)

// LogTailer tails Factorio server pod logs and fans out parsed events to subscribers.
type LogTailer struct {
	podLabel    string
	k8s         *K8sClient
	lastPod     string
	subscribers []LogSubscriber
}

func NewLogTailer(podLabel string, k8s *K8sClient) *LogTailer {
	return &LogTailer{
		podLabel: podLabel,
		k8s:      k8s,
	}
}

func (t *LogTailer) Subscribe(sub LogSubscriber) {
	t.subscribers = append(t.subscribers, sub)
}

func (t *LogTailer) Run(ctx context.Context) {
	for {
		if err := t.tail(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("log tail error: %v, retrying in 10s", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

func (t *LogTailer) tail(ctx context.Context) error {
	podName, err := t.k8s.FindPod(ctx, t.podLabel)
	if err != nil {
		return fmt.Errorf("find pod: %w", err)
	}

	if t.lastPod != podName {
		log.Printf("tailing logs from pod %s/%s", t.k8s.namespace, podName)
		t.lastPod = podName
	}

	body, err := t.k8s.StreamLogs(ctx, podName)
	if err != nil {
		return fmt.Errorf("stream logs: %w", err)
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}
		t.parseLine(scanner.Text())
	}
	return scanner.Err()
}

func (t *LogTailer) parseLine(line string) {
	now := time.Now()
	var event *GameEvent

	if m := chatPattern.FindStringSubmatch(line); m != nil {
		event = &GameEvent{Type: "chat", Player: m[1], Message: m[2], Time: now}
	} else if m := joinPattern.FindStringSubmatch(line); m != nil {
		event = &GameEvent{Type: "join", Player: m[1], Time: now}
	} else if m := leavePattern.FindStringSubmatch(line); m != nil {
		event = &GameEvent{Type: "leave", Player: m[1], Time: now}
	} else if m := researchPattern.FindStringSubmatch(line); m != nil {
		event = &GameEvent{Type: "research", Extra: map[string]string{"tech": m[1]}, Time: now}
	} else if rocketPattern.MatchString(line) {
		event = &GameEvent{Type: "rocket", Time: now}
	} else if m := savePattern.FindStringSubmatch(line); m != nil {
		event = &GameEvent{Type: "save", Extra: map[string]string{"name": m[1]}, Time: now}
	}

	if event != nil {
		for _, sub := range t.subscribers {
			sub.OnLogEvent(*event)
		}
	}
}
