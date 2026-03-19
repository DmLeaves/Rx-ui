package main

import "testing"

func TestParseStatOutput_Multiline(t *testing.T) {
	input := `stat: <
  name: "inbound>>>demo_tag>>>traffic>>>uplink"
  value: 12345
>
stat: <
  name: "inbound>>>demo_tag>>>traffic>>>downlink"
  value: 67890
>`

	m := parseStatOutput(input)
	if m["inbound>>>demo_tag>>>traffic>>>uplink"] != 12345 {
		t.Fatalf("uplink parse failed: %#v", m)
	}
	if m["inbound>>>demo_tag>>>traffic>>>downlink"] != 67890 {
		t.Fatalf("downlink parse failed: %#v", m)
	}
}

func TestParseStatOutput_SingleLine(t *testing.T) {
	input := `name: "inbound>>>demo>>>traffic>>>uplink" value: 111
name: "inbound>>>demo>>>traffic>>>downlink" value: 222`

	m := parseStatOutput(input)
	if m["inbound>>>demo>>>traffic>>>uplink"] != 111 || m["inbound>>>demo>>>traffic>>>downlink"] != 222 {
		t.Fatalf("single-line parse failed: %#v", m)
	}
}
