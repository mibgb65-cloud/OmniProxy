package activity

import (
	"context"
	"testing"
	"time"
)

// 进行中的长请求不能被随后完成的请求挤到中间：列表永远把 pending 排在最前。
func TestRequestsPinPendingRecordsFirst(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()

	// pending 只写 Begin + Update：Finish 会把它标成已结束。
	pending := event(9, "member@example.invalid", "")
	pending.State, pending.StatusCode = "pending", 0
	if err := s.Begin(ctx, pending); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, pending); err != nil {
		t.Fatal(err)
	}

	// 三条在它之后才开始、并且已经完成的请求。
	for _, id := range []int{6, 7, 8} {
		finished := event(id, "member@example.invalid", "")
		finished.StartedAt = pending.StartedAt.Add(time.Duration(id-5) * time.Second)
		record(t, s, finished)
	}

	page, err := s.Requests(ctx, RequestQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 4 {
		t.Fatalf("items = %d, want 4", len(page.Items))
	}
	if page.Items[0].State != "pending" || page.Items[0].ID != pending.ID {
		t.Fatalf("first item = %q/%s, want the pending record", page.Items[0].State, page.Items[0].ID)
	}
	// 已结束的一组仍按开始时间倒序。
	for index := 2; index < len(page.Items); index++ {
		if page.Items[index-1].StartedAt < page.Items[index].StartedAt {
			t.Fatalf("finished order broke at %d: %+v", index, page.Items)
		}
	}
	if page.Items[1].ID != event(8, "", "").ID {
		t.Fatalf("newest finished record = %s", page.Items[1].ID)
	}

	// 只看已结束时排序仍按时间，不受 pending 优先级影响。
	filtered, err := s.Requests(ctx, RequestQuery{State: "success", Limit: 10})
	if err != nil || len(filtered.Items) != 3 {
		t.Fatalf("filtered = %d, %v", len(filtered.Items), err)
	}
	for _, item := range filtered.Items {
		if item.State != "success" {
			t.Fatalf("state filter leaked %q", item.State)
		}
	}
}
