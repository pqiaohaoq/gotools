package timewheel

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_getPositionAndCircle(t *testing.T) {
	tables := []struct {
		delay    int
		interval int
		slotNum  int
		current  int
		position int
		circle   int
	}{
		{delay: 100, interval: 100, slotNum: 10, position: 1, circle: 0},
		{delay: 101, interval: 100, slotNum: 10, position: 2, circle: 0},
		{delay: 100, interval: 100, slotNum: 10, position: 0, circle: 0, current: 9},
		{delay: 1000, interval: 100, slotNum: 10, position: 0, circle: 1},
		{delay: 2000, interval: 100, slotNum: 10, position: 0, circle: 2},
		{delay: 1001, interval: 100, slotNum: 10, position: 1, circle: 1},
	}

	for _, table := range tables {

		tw := &TimeWheel{slotNum: table.slotNum, tickInterval: time.Duration(table.interval) * time.Millisecond, currentPosition: table.current}
		p, c := tw.getPositionAndCircle(time.Duration(table.delay) * time.Millisecond)
		fmt.Println("delay", table.delay, "position", table.position, "circle", table.circle)
		assert.Equal(t, table.position, p)
		assert.Equal(t, table.circle, c)
	}
}
