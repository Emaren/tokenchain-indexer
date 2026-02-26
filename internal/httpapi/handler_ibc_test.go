package httpapi

import "testing"

func TestNormalizeEnumValue(t *testing.T) {
	if got := normalizeEnumValue("STATE_OPEN", "STATE_"); got != "OPEN" {
		t.Fatalf("expected OPEN, got %q", got)
	}
	if got := normalizeEnumValue(" ORDER_UNORDERED ", "ORDER_"); got != "UNORDERED" {
		t.Fatalf("expected UNORDERED, got %q", got)
	}
}

func TestToIBCChannelItem(t *testing.T) {
	channel := chainIBCChannel{
		State:          "STATE_OPEN",
		Ordering:       "ORDER_UNORDERED",
		ConnectionHops: []string{"connection-7"},
		Version:        "ics20-1",
		PortID:         "transfer",
		ChannelID:      "channel-9",
	}
	channel.Counterparty.PortID = "transfer"
	channel.Counterparty.ChannelID = "channel-42"

	item := toIBCChannelItem(channel)
	if item.State != "OPEN" {
		t.Fatalf("expected state OPEN, got %q", item.State)
	}
	if item.Ordering != "UNORDERED" {
		t.Fatalf("expected ordering UNORDERED, got %q", item.Ordering)
	}
	if item.ConnectionID != "connection-7" {
		t.Fatalf("expected connection-7, got %q", item.ConnectionID)
	}
	if item.CounterpartyChannelID != "channel-42" {
		t.Fatalf("expected channel-42, got %q", item.CounterpartyChannelID)
	}
}
