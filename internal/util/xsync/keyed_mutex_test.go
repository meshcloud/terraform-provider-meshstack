package xsync

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyedMutex(t *testing.T) {
	t.Run("serializes a read-modify-write on one key", func(t *testing.T) {
		var m KeyedMutex
		tags := map[string]int{}
		var wg sync.WaitGroup
		for range 100 {
			wg.Go(func() {
				defer m.Lock("key")()
				read := tags["count"]
				tags["count"] = read + 1
			})
		}
		wg.Wait()
		assert.Equal(t, 100, tags["count"])
	})

	t.Run("keeps different keys independent", func(t *testing.T) {
		var m KeyedMutex
		unlockFirst := m.Lock("first")
		defer unlockFirst()
		done := make(chan struct{})
		go func() {
			defer m.Lock("second")()
			close(done)
		}()
		<-done
	})
}
