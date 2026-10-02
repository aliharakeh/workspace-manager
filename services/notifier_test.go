package services

import "testing"

func TestNotifyEnabledDefaultsOn(t *testing.T) {
	settings := map[string]string{}
	for _, kind := range []string{NotifyReady, NotifyFinished, NotifyError, NotifyIdle} {
		if !notifyEnabled(settings, kind) {
			t.Fatalf("%s should default to enabled", kind)
		}
	}
}

func TestNotifyEnabledMasterSwitch(t *testing.T) {
	settings := map[string]string{NotifyKeyEnabled: "0"}
	for _, kind := range []string{NotifyReady, NotifyFinished, NotifyError, NotifyIdle} {
		if notifyEnabled(settings, kind) {
			t.Fatalf("master switch off should disable %s", kind)
		}
	}
}

func TestNotifyEnabledPerKind(t *testing.T) {
	settings := map[string]string{NotifyKeyError: "0"}
	if notifyEnabled(settings, NotifyError) {
		t.Fatal("error notifications should be disabled")
	}
	if !notifyEnabled(settings, NotifyReady) {
		t.Fatal("ready notifications should stay enabled")
	}
}

func TestIdleSeconds(t *testing.T) {
	cases := map[string]int{
		"":     DefaultIdleSeconds,
		"0":    DefaultIdleSeconds,
		"-5":   DefaultIdleSeconds,
		"abc":  DefaultIdleSeconds,
		"30":   30,
		"9999": 600,
	}
	for raw, want := range cases {
		if got := idleSeconds(map[string]string{NotifyKeyIdleSecs: raw}); got != want {
			t.Fatalf("idleSeconds(%q) = %d, want %d", raw, got, want)
		}
	}
}
