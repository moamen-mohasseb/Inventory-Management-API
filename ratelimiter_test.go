package main

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsBurstThenBlocks(t *testing.T) {
	rl := NewRateLimiter(1, 5)
	now := time.Now()

	for i := 1; i <= 5; i++ {
		if ok, _ := rl.allowAt("client", now); !ok {
			t.Fatalf("request %d should be allowed (burst of 5)", i)
		}
	}

	ok, wait := rl.allowAt("client", now)
	if ok {
		t.Fatal("6th request in the same instant should be rejected")
	}
	if wait <= 0 {
		t.Fatal("a rejected request must report a positive wait time")
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	rl := NewRateLimiter(1, 5)
	now := time.Now()

	for i := 0; i < 5; i++ {
		rl.allowAt("client", now)
	}

	if ok, _ := rl.allowAt("client", now.Add(time.Second)); !ok {
		t.Fatal("one token should have been refilled after one second")
	}
	if ok, _ := rl.allowAt("client", now.Add(time.Second)); ok {
		t.Fatal("only one token should have been refilled after one second")
	}
}

func TestRateLimiterTracksClientsSeparately(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	now := time.Now()

	if ok, _ := rl.allowAt("a", now); !ok {
		t.Fatal("first request of client a should be allowed")
	}
	if ok, _ := rl.allowAt("a", now); ok {
		t.Fatal("second request of client a should be rejected")
	}
	if ok, _ := rl.allowAt("b", now); !ok {
		t.Fatal("client b has its own bucket and should be allowed")
	}
}
