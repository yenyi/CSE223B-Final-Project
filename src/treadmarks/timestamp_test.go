package src

import (
	"runtime/debug"
	"testing"
)

func TestTimestamp_increment(t *testing.T) {
	ts1 := Timestamp{0, 0, 0}
	ts1 = ts1.increment(1)

	trueEqual := func(b bool) {
		if b != true {
			debug.PrintStack()
			t.Fatal()
		}
	}
	trueEqual(ts1.equals(Timestamp{0, 1, 0}))

}

func TestTimestamp_cover(t *testing.T) {
	trueEqual := func(b bool) {
		if b != true {
			debug.PrintStack()
			t.Fatal()
		}
	}
	falseEqual := func(b bool) {
		if b != false {
			debug.PrintStack()
			t.Fatal()
		}
	}
	ts1 := Timestamp{0, 0, 0}
	ts2 := Timestamp{1, 0, 0}
	ts3 := Timestamp{0, 0, 1}
	trueEqual(ts1.larger(ts1))
	falseEqual(ts1.larger(ts2))
	falseEqual(ts1.larger(ts3))
	trueEqual(ts2.larger(ts1))
	trueEqual(ts2.larger(ts2))
	falseEqual(ts2.larger(ts3))
	trueEqual(ts3.larger(ts1))
	falseEqual(ts3.larger(ts2))
	trueEqual(ts3.larger(ts3))
}

func TestTimestamp_loadsave(t *testing.T) {
	valueEqual := func(a interface{}, b interface{}) {
		if a != b {
			debug.PrintStack()
			t.Fatal()
		}
	}
	timestampEqual := func(a Timestamp, b Timestamp) {

		if len(a) != len(b) {
			debug.PrintStack()
			t.Fatal()
		}
		for i := range a {
			if a[i] != b[i] {
				debug.PrintStack()
				t.Fatal()
			}
		}
	}
	ts1 := Timestamp{0, 0, 0}
	data := ts1.saveToData()
	for i := range data {
		valueEqual(byte(0), data[i])
	}
	ts1.increment(1)
	data = ts1.saveToData()
	ts2 := Timestamp{0, 0, 0}
	ts2.loadFromData(data)
	timestampEqual(ts1, ts2)

}
