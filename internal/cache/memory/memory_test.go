package memory

import (
	"context"
	"github.com/mylxsw/go-utils/assert"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	memory := New(context.TODO())

	{
		assert.NoError(t, memory.Set(context.TODO(), "abc", "123"))
		res, err := memory.Get(context.TODO(), "abc")
		assert.NoError(t, err)
		assert.Equal(t, "123", res)
	}

	{
		assert.NoError(t, memory.Expire(context.TODO(), "abc", 30*time.Second))
		ttl, err := memory.TTL(context.TODO(), "abc")
		assert.NoError(t, err)
		assert.True(t, ttl.Seconds() > 20 && ttl.Seconds() < 30)
	}

	{
		assert.NoError(t, memory.Incr(context.TODO(), "abc"))
		res, err := memory.Get(context.TODO(), "abc")
		assert.NoError(t, err)
		assert.Equal(t, "124", res)
	}

	{
		assert.NoError(t, memory.Incr(context.TODO(), "efg"))
		res, err := memory.Get(context.TODO(), "efg")
		assert.NoError(t, err)
		assert.Equal(t, "1", res)
	}
}
