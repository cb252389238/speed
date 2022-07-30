package speed

import (
	"fmt"
	"testing"
)

func TestSpeedCache(t *testing.T) {
	c, err := New()
	fmt.Println(c, err)
}
