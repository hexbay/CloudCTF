package httpserver

import "testing"

func TestUpdateJobStatePreventsConcurrentUpdates(t *testing.T) {
	state := &updateJobState{}
	first, started := state.begin("v1.2.3", "v1.2.2")
	if !started {
		t.Fatal("expected first update job to start")
	}
	if _, started := state.begin("v1.2.4", "v1.2.2"); started {
		t.Fatal("expected concurrent update job to be rejected")
	}

	state.update(first.ID, func(job *updateJobSnapshot) {
		job.Stage = "download"
		job.Message = "downloading"
	})
	current := state.current()
	if current == nil || current.Stage != "download" || current.Message != "downloading" {
		t.Fatalf("unexpected update job state: %#v", current)
	}
}

func TestUpdateJobStateAllowsRetryAfterFailure(t *testing.T) {
	state := &updateJobState{}
	first, started := state.begin("v1.2.3", "v1.2.2")
	if !started {
		t.Fatal("expected first update job to start")
	}
	state.update(first.ID, func(job *updateJobSnapshot) {
		job.Status = "failed"
		job.Error = "download failed"
	})

	second, started := state.begin("v1.2.3", "v1.2.2")
	if !started || second.ID == first.ID {
		t.Fatalf("expected a new update job after failure, got %#v", second)
	}
}
