package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetXrayStats_WithMockXrayApi(t *testing.T) {
	tmp := t.TempDir()
	mock := filepath.Join(tmp, "xray-mock.sh")
	script := `#!/usr/bin/env bash
if [[ "$1" == "api" && "$2" == "statsquery" ]]; then
cat <<'OUT'
stat: <
  name: "inbound>>>demo_tag>>>traffic>>>uplink"
  value: 333
>
stat: <
  name: "inbound>>>demo_tag>>>traffic>>>downlink"
  value: 666
>
OUT
exit 0
fi
exit 1
`
	if err := os.WriteFile(mock, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	oldPath := settings["xrayBinPath"]
	oldRun := xrayRunning
	settings["xrayBinPath"] = mock
	xrayRunning = true
	defer func() {
		settings["xrayBinPath"] = oldPath
		xrayRunning = oldRun
	}()

	stats, err := getXrayStats()
	if err != nil {
		t.Fatalf("getXrayStats err: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d: %#v", len(stats), stats)
	}
	if stats[0].Tag != "demo_tag" || stats[0].Uplink != 333 || stats[0].Downlink != 666 {
		t.Fatalf("unexpected stats: %#v", stats[0])
	}
}
