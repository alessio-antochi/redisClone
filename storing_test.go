package main

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestSetThenGet(t *testing.T) {
	s := NewStore()
	s.Set("foo", []byte("bar"))
	got, ok := s.Get("foo")
	if !ok || string(got) != "bar" {
		t.Fatalf("got %q, ok=%v; want \"bar\", true", got, ok)
	}
}

func TestStoreConcurrency(t *testing.T) {
	store := NewStore()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func (n int) {
			defer wg.Done()

			key := strconv.Itoa(n % 5) // force collision
			value := []byte(strconv.Itoa(n))

			store.Set(key, value)
			store.Get(key)
			store.Exists(key)
			store.Del(key)
		}(i)
	}

	wg.Wait()
}

func TestNoLostUpdateOnExpiryRace(t *testing.T) {
	for range 5000 {
		store := NewStore()
		key := "race-key"

		store.Set(key, []byte("stale"))
		store.Expire(key, 1*time.Nanosecond)
		time.Sleep(time.Microsecond) // make sure its expired

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			store.Get(key)
		}()

		go func() {
			defer wg.Done()
			store.Set(key, []byte("fresh"))
		}()

		wg.Wait()

		val, ok := store.Get(key)
		if !ok {
			t.Fatalf("Key missing after concurrent SET -- lost update")
		}
		if string(val) != "fresh" {
			t.Fatalf("Got %q, want %q", val, "fresh")
		}
	}
}
