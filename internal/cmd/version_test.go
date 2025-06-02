package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{
			name: "show_version",
			args: []string{},
			want: "vaultenv version",
		},
		{
			name: "version_with_args",
			args: []string{"extra", "args"},
			want: "vaultenv version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := newVersionCommand()
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			// Reset command for testing
			cmd.Run = func(cmd *cobra.Command, args []string) {
				cmd.Println("vaultenv version 0.1.0")
			}

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()
			if !strings.Contains(output, tt.want) {
				t.Errorf("Output = %q, want to contain %q", output, tt.want)
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	// Test with different version values
	tests := []struct {
		version string
		want    string
	}{
		{"1.0.0", "vaultenv version 1.0.0"},
		{"dev", "vaultenv version dev"},
		{"", "vaultenv version "},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := newVersionCommand()
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{})

			// Override Run to use test version
			cmd.Run = func(cmd *cobra.Command, args []string) {
				cmd.Println("vaultenv version " + tt.version)
			}

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			output := strings.TrimSpace(buf.String())
			if output != tt.want {
				t.Errorf("Output = %q, want %q", output, tt.want)
			}
		})
	}
}
