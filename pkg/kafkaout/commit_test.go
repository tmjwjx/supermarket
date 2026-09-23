package kafkaout

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func noDelay(int) time.Duration { return 0 }

type collector struct{ batches [][]*kgo.Record }

func (c *collector) commit(recs []*kgo.Record) {
	c.batches = append(c.batches, append([]*kgo.Record(nil), recs...))
}

func (c *collector) all() []string {
	var out []string
	for _, batch := range c.batches {
		out = append(out, keys(batch)...)
	}
	return out
}

// 第一次失败第二次成功 同一会话内原地重试 该条和后续记录都要处理并提交
func TestConsumeRetriesInPlaceUntilSuccess(t *testing.T) {
	recs := []*kgo.Record{
		{Topic: "t", Partition: 0, Offset: 2, Key: []byte("b")},
		{Topic: "t", Partition: 0, Offset: 1, Key: []byte("a")},
	}
	calls := map[string]int{}
	var seq []string
	c := &collector{}
	consumeBatch(context.Background(), recs, func(_ context.Context, ev Event) error {
		calls[ev.Key]++
		seq = append(seq, ev.Key)
		if ev.Key == "a" && calls["a"] == 1 {
			return errors.New("boom")
		}
		return nil
	}, noDelay, c.commit)
	if calls["a"] != 2 || calls["b"] != 1 {
		t.Fatalf("calls %v", calls)
	}
	if got := join(seq); got != "a,a,b" {
		t.Fatalf("order %s", got)
	}
	if got := join(c.all()); got != "a,b" {
		t.Fatalf("commit %s", got)
	}
}

// 连续失败满 5 次记日志跳过 之后的记录照常处理 两条都提交
func TestConsumeSkipsAfterFiveFailures(t *testing.T) {
	recs := []*kgo.Record{
		{Topic: "t", Partition: 0, Offset: 1, Key: []byte("a")},
		{Topic: "t", Partition: 0, Offset: 2, Key: []byte("b")},
	}
	calls := map[string]int{}
	c := &collector{}
	consumeBatch(context.Background(), recs, func(_ context.Context, ev Event) error {
		calls[ev.Key]++
		if ev.Key == "a" {
			return errors.New("boom")
		}
		return nil
	}, noDelay, c.commit)
	if calls["a"] != maxAttempts || calls["b"] != 1 {
		t.Fatalf("calls %v", calls)
	}
	if got := join(c.all()); got != "a,b" {
		t.Fatalf("commit %s", got)
	}
}

// 重试等待中停机 失败的那条和它后面的都不提交也不处理 别的分区已处理完的照常提交
func TestConsumeStopsPartitionOnShutdown(t *testing.T) {
	recs := []*kgo.Record{
		{Topic: "t", Partition: 1, Offset: 1, Key: []byte("c")},
		{Topic: "t", Partition: 0, Offset: 1, Key: []byte("a")},
		{Topic: "t", Partition: 0, Offset: 2, Key: []byte("b")},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := map[string]int{}
	c := &collector{}
	consumeBatch(ctx, recs, func(_ context.Context, ev Event) error {
		calls[ev.Key]++
		if ev.Key == "a" {
			cancel()
			return errors.New("boom")
		}
		return nil
	}, func(int) time.Duration { return time.Hour }, c.commit)
	if calls["a"] != 1 || calls["b"] != 0 {
		t.Fatalf("calls %v", calls)
	}
	if got := join(c.all()); got != "c" {
		t.Fatalf("commit %s", got)
	}
}

// 每个分区单独提交一次
func TestConsumeCommitsPerPartition(t *testing.T) {
	recs := []*kgo.Record{
		{Topic: "t", Partition: 0, Offset: 1, Key: []byte("a")},
		{Topic: "t", Partition: 1, Offset: 1, Key: []byte("c")},
		{Topic: "t", Partition: 0, Offset: 2, Key: []byte("b")},
	}
	c := &collector{}
	consumeBatch(context.Background(), recs, func(context.Context, Event) error { return nil }, noDelay, c.commit)
	if len(c.batches) != 2 || join(keys(c.batches[0])) != "a,b" || join(keys(c.batches[1])) != "c" {
		t.Fatalf("batches %v", c.batches)
	}
}

// 没有 event_id 头时按主题分区偏移生成 同一业务键的两条事件不能撞 id
func TestEventIDFallsBackToOffset(t *testing.T) {
	first := eventFrom(&kgo.Record{Topic: "order.paid", Partition: 3, Offset: 7, Key: []byte("o1")})
	second := eventFrom(&kgo.Record{Topic: "order.paid", Partition: 3, Offset: 8, Key: []byte("o1")})
	if first.ID != "order.paid/3/7" || second.ID != "order.paid/3/8" {
		t.Fatalf("ids %s %s", first.ID, second.ID)
	}
	withHeader := eventFrom(&kgo.Record{Topic: "t", Key: []byte("o1"), Headers: []kgo.RecordHeader{{Key: "event_id", Value: []byte("e1")}}})
	if withHeader.ID != "e1" {
		t.Fatalf("id %s", withHeader.ID)
	}
}

func TestRetryDelayBackoff(t *testing.T) {
	want := []time.Duration{200 * time.Millisecond, 400 * time.Millisecond, 800 * time.Millisecond, 1600 * time.Millisecond, 3 * time.Second, 3 * time.Second}
	for i, w := range want {
		if got := retryDelay(i + 1); got != w {
			t.Fatalf("attempt %d delay %v", i+1, got)
		}
	}
}

func keys(recs []*kgo.Record) []string {
	out := make([]string, 0, len(recs))
	for _, rec := range recs {
		out = append(out, string(rec.Key))
	}
	return out
}

func join(items []string) string { return strings.Join(items, ",") }
