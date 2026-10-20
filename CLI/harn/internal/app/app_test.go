package app

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestVersionAndHelp(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	runner := New(Options{Stdout: &stdout, Stderr: &stderr, Version: "v1.2.3"})
	if code := runner.Run(context.Background(), []string{"version"}); code != 0 {
		t.Fatalf("version exit code %d", code)
	}
	if stdout.String() != "harn v1.2.3\n" {
		t.Fatalf("unexpected output %q", stdout.String())
	}

	stdout.Reset()
	if code := runner.Run(context.Background(), []string{"help"}); code != 0 {
		t.Fatalf("help exit code %d", code)
	}
	if !strings.Contains(stdout.String(), "install") {
		t.Fatalf("help does not describe install: %q", stdout.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	runner := New(Options{Stdout: &stdout, Stderr: &stderr})
	if code := runner.Run(context.Background(), []string{"unknown"}); code != 2 {
		t.Fatalf("got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), `unknown command "unknown"`) {
		t.Fatalf("unexpected error %q", stderr.String())
	}
}

func TestInstallRequiresArguments(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	runner := New(Options{Stdout: &stdout, Stderr: &stderr})
	if code := runner.Run(context.Background(), []string{"install"}); code != 2 {
		t.Fatalf("got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "requires --source") {
		t.Fatalf("unexpected error %q", stderr.String())
	}
}

func TestUpdateRequiresPathOrAll(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	runner := New(Options{Stdout: &stdout, Stderr: &stderr})
	if code := runner.Run(context.Background(), []string{"update"}); code != 2 {
		t.Fatalf("got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "requires --path or --all") {
		t.Fatalf("unexpected error %q", stderr.String())
	}
}

func TestUpdateRejectsPathWithAll(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	runner := New(Options{Stdout: &stdout, Stderr: &stderr})
	if code := runner.Run(context.Background(), []string{"update", "--all", "--path", "Skills/swe"}); code != 2 {
		t.Fatalf("got exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "not both") {
		t.Fatalf("unexpected error %q", stderr.String())
	}
}

func TestListAndStatusAndRemoveRequireRecording(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	var stdout, stderr bytes.Buffer
	runner := New(Options{Stdout: &stdout, Stderr: &stderr})

	if code := runner.Run(context.Background(), []string{"list", "--root", project}); code != 0 {
		t.Fatalf("list exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nothing recorded") {
		t.Fatalf("unexpected list output %q", stdout.String())
	}

	stdout.Reset()
	if code := runner.Run(context.Background(), []string{"status", "--root", project}); code != 0 {
		t.Fatalf("status exit code %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "nothing recorded") {
		t.Fatalf("unexpected status output %q", stdout.String())
	}

	stdout.Reset()
	if code := runner.Run(context.Background(), []string{"remove", "--root", project}); code != 2 {
		t.Fatalf("remove without path exit code %d", code)
	}
	if !strings.Contains(stderr.String(), "requires at least one --path") {
		t.Fatalf("unexpected error %q", stderr.String())
	}
}

func TestHelpListsManagementCommands(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	runner := New(Options{Stdout: &stdout, Stderr: &stderr})
	if code := runner.Run(context.Background(), []string{"help"}); code != 0 {
		t.Fatalf("help exit code %d", code)
	}
	for _, command := range []string{"install", "update", "list", "status", "remove", "version"} {
		if !strings.Contains(stdout.String(), command) {
			t.Fatalf("help is missing %s: %q", command, stdout.String())
		}
	}
}

func TestSearchAcceptsQueryBeforeFlags(t *testing.T) {
	t.Parallel()
	reordered := floatFlagsFirst([]string{"review", "--source", "https://example.test", "--version", "tag:v1"})
	want := []string{"--source", "https://example.test", "--version", "tag:v1", "review"}
	if strings.Join(reordered, " ") != strings.Join(want, " ") {
		t.Fatalf("got %#v", reordered)
	}

	equals := floatFlagsFirst([]string{"review", "--source=https://example.test"})
	want = []string{"--source=https://example.test", "review"}
	if strings.Join(equals, " ") != strings.Join(want, " ") {
		t.Fatalf("got %#v", equals)
	}
}
