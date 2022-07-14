package timewheel

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func newTask(delay time.Duration, t *testing.T) *Task {
	tt := &Task{
		TimeoutCallback: func(task Task) {
			if task.Elasped() < delay {
				t.Error("time out value is error, it should be larger than specific time out")
			}
		},
	}

	return tt
}

func Test_TimeWheelOptions(t *testing.T) {
	convey.Convey("Test function options", t, func() {
		convey.Convey("test default options", func() {
			timewheel, err := New(time.Second, 1)
			convey.So(err, convey.ShouldBeNil)
			convey.So(timewheel.taskConcurrent, convey.ShouldEqual, defaultTaskConcurrent)
			convey.So(defaultOptions.TaskConcurrent, convey.ShouldEqual, defaultTaskConcurrent)
		})

		convey.Convey("modify the default options", func() {
			timewheel, err := New(time.Second, 1, WithTaskConcurrent(100))
			convey.So(err, convey.ShouldBeNil)
			convey.So(timewheel.taskConcurrent, convey.ShouldEqual, 100)
			convey.So(defaultOptions.TaskConcurrent, convey.ShouldEqual, defaultTaskConcurrent)
		})
	})
}

func Test_TimeWheelStartAndTasks(t *testing.T) {
	convey.Convey("New time wheel success and start", t, func() {
		slotNum := 16
		interval := 100 * time.Millisecond

		timewheel, err := New(interval, slotNum)
		convey.So(err, convey.ShouldBeNil)
		convey.So(timewheel, convey.ShouldNotBeNil)

		timewheel.Start()
		defer timewheel.Stop()

		convey.So(timewheel.ticker, convey.ShouldNotBeNil)
		convey.So(timewheel.taskGoroutine, convey.ShouldNotBeNil)

		convey.Convey("add time task with error parameter", func() {
			delay := time.Duration(0)
			tt := newTask(delay, t)
			err := timewheel.AddTask(delay, "1", tt)
			convey.So(err, convey.ShouldBeError)

			delay = interval / 2
			tt = newTask(delay, t)
			err = timewheel.AddTask(delay, "1", tt)
			convey.So(err, convey.ShouldBeError)

			delay = interval * 2
			tt = newTask(delay, t)
			err = timewheel.AddTask(delay, "", tt)
			convey.So(err, convey.ShouldBeError)
		})

		var id int32 = 0
		convey.Convey("add a time task", func() {
			key := strconv.Itoa(int(atomic.AddInt32(&id, 1)))
			delay := 1 * time.Second
			tt := newTask(delay, t)
			err := timewheel.AddTask(delay, key, tt)
			convey.So(err, convey.ShouldBeNil)

			exist := timewheel.HasTask(key)
			convey.So(exist, convey.ShouldBeTrue)

			time.Sleep(1*time.Second + timewheel.interval + 50*time.Millisecond)

			exist = timewheel.HasTask(key)
			convey.So(exist, convey.ShouldBeFalse)
		})
	})
}
