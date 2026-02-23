package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rachanaanugandula/kube-pfs/pkg/metrics"
)

type event struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Status    string `json:"status"`
	Detail    string `json:"detail"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: fault-injector <delete-pod|netem-delay|corrupt-block> [flags]")
		os.Exit(1)
	}

	action := os.Args[1]
	args := os.Args[2:]

	logPath := "./artifacts/faults/timeline.jsonl"
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--timeline" {
			logPath = args[i+1]
		}
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "create timeline dir: %v\n", err)
		os.Exit(1)
	}

	var runErr error
	switch action {
	case "delete-pod":
		runErr = runDeletePod(args)
	case "netem-delay":
		runErr = runNetemDelay(args)
	case "corrupt-block":
		runErr = runCorruptBlock(args)
	default:
		runErr = fmt.Errorf("unknown action: %s", action)
	}

	now := time.Now().UTC()
	status := "ok"
	detail := action + " completed"
	if runErr != nil {
		status = "error"
		detail = runErr.Error()
	}
	if err := appendEvent(logPath, event{Timestamp: now.Format(time.RFC3339), Action: action, Status: status, Detail: detail}); err != nil {
		fmt.Fprintf(os.Stderr, "append timeline event: %v\n", err)
	}
	metrics.RecordFaultEvent(action, status, now)

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "%s failed: %v\n", action, runErr)
		os.Exit(1)
	}
	fmt.Printf("%s succeeded\n", action)
}

func runDeletePod(args []string) error {
	fs := flag.NewFlagSet("delete-pod", flag.ContinueOnError)
	namespace := fs.String("namespace", "kube-pfs-test", "pod namespace")
	name := fs.String("name", "", "pod name")
	_ = fs.String("timeline", "", "timeline path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return errors.New("--name is required")
	}
	return runCommand("kubectl", "delete", "pod", *name, "-n", *namespace, "--wait=false")
}

func runNetemDelay(args []string) error {
	fs := flag.NewFlagSet("netem-delay", flag.ContinueOnError)
	namespace := fs.String("namespace", "kube-pfs-test", "pod namespace")
	pod := fs.String("pod", "", "pod name")
	iface := fs.String("interface", "eth0", "network interface")
	delay := fs.String("delay", "200ms", "netem delay")
	_ = fs.String("timeline", "", "timeline path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*pod) == "" {
		return errors.New("--pod is required")
	}
	return runCommand("kubectl", "exec", "-n", *namespace, *pod, "--", "tc", "qdisc", "replace", "dev", *iface, "root", "netem", "delay", *delay)
}

func runCorruptBlock(args []string) error {
	fs := flag.NewFlagSet("corrupt-block", flag.ContinueOnError)
	path := fs.String("path", "", "path to block file")
	bytesCount := fs.Int("bytes", 256, "number of random bytes to write")
	_ = fs.String("timeline", "", "timeline path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*path) == "" {
		return errors.New("--path is required")
	}
	if *bytesCount <= 0 {
		return errors.New("--bytes must be > 0")
	}
	f, err := os.OpenFile(*path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	payload := make([]byte, *bytesCount)
	for i := range payload {
		payload[i] = byte((i*17 + 31) % 255)
	}
	_, err = f.WriteAt(payload, 0)
	return err
}

func appendEvent(path string, e event) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	blob, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := w.WriteString(string(blob) + "\n"); err != nil {
		return err
	}
	return w.Flush()
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
