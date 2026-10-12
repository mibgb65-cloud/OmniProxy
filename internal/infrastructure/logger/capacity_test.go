package logger

import "testing"

func TestRequestedLogCapacity5000(t *testing.T) {
	buffer := NewUIBuffer(10000)
	for i := 0; i < 5002; i++ {
		buffer.Add("safe log")
	}
	snapshot, err := buffer.Read(Query{Limit: 200})
	if err != nil || snapshot.Capacity != 5000 || buffer.Len() != 5000 || snapshot.OldestID != 3 {
		t.Fatal("log cap mismatch", snapshot.Capacity, err)
	}
}
