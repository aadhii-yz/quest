package runner

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aadhii-yz/quest/internal/provider"
)

func collect(t *testing.T, ch <-chan Event) []string {
	t.Helper()
	var lines []string
	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatal("channel closed without a done event")
			}
			if ev.Done {
				return lines
			}
			lines = append(lines, ev.Line)
		case <-timeout:
			t.Fatal("timed out waiting for events")
		}
	}
}

func request() Request {
	return Request{
		CourseID: "grep",
		Stage:    provider.Stage{ID: "cq2", Name: "Match a literal character", Number: 1},
		Language: "go",
		Tester: provider.TesterPlan{
			Available: true, Platform: "linux_amd64",
			AssetName: "v65_linux_amd64.tar.gz", Release: "v65",
			License: provider.LicenseNone,
		},
	}
}

func TestSimulatedOrderAndLabels(t *testing.T) {
	events, err := Simulated{}.Run(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	lines := collect(t, events)
	if len(lines) == 0 {
		t.Fatal("no events")
	}
	if !strings.Contains(lines[0], "cq2") || !strings.Contains(lines[0], "go") {
		t.Fatalf("first line = %q", lines[0])
	}
	if !strings.Contains(strings.Join(lines, "\n"), "simulated runner") {
		t.Fatal("simulated output must be labelled")
	}
	var pass, warning bool
	for _, line := range lines {
		pass = pass || strings.HasPrefix(line, "PASS")
		warning = warning || strings.Contains(line, "no license")
	}
	if !pass || !warning {
		t.Fatalf("missing pass or license warning: %v", lines)
	}
}

func TestSimulatedMissingPlatform(t *testing.T) {
	req := request()
	req.Tester = provider.TesterPlan{Available: false, Platform: "darwin_arm64", License: provider.LicenseNone}
	events, err := Simulated{}.Run(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	lines := collect(t, events)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "darwin_arm64") {
		t.Fatalf("missing platform in output: %v", lines)
	}
	if last := lines[len(lines)-1]; !strings.HasPrefix(last, "FAIL") {
		t.Fatalf("last line = %q", last)
	}
}

func TestSimulatedCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	events, err := Simulated{Delay: time.Hour}.Run(ctx, request())
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range events {
		}
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runner did not stop after cancellation")
	}
}
