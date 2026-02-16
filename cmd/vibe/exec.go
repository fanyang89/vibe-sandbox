package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var (
	runCommandFn         = runCommand
	commandOutputFn      = commandOutput
	commandOutputNoErrFn = commandOutputNoErr
	gitOutputFn          = gitOutput
	interactiveCommandFn = runInteractiveCommand
)

func runCommand(dir string, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func commandOutput(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func commandOutputNoErr(dir, name string, args ...string) string {
	out, _ := commandOutput(dir, name, args...)
	return out
}

func gitOutput(dir string, args ...string) (string, error) {
	out, err := commandOutput(dir, "git", args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func runInteractiveCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
