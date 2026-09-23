package kafkaout

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
	"gorm.io/gorm"
)

// seedBrokers 由 UseBrokers 在启动时写入 之后只读
var seedBrokers []string

// UseBrokers 把配置文件里的地址写进来 多个地址用英文逗号分开 空白表示未配置
func UseBrokers(raw string) {
	seedBrokers = splitBrokers(raw)
}

// Brokers 返回 UseBrokers 写入的地址 未配置时为空
func Brokers() []string {
	if len(seedBrokers) == 0 {
		return nil
	}
	out := make([]string, len(seedBrokers))
	copy(out, seedBrokers)
	return out
}

func splitBrokers(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// Event 是发件箱里的一条待投递事件
type Event struct {
	ID      string
	Topic   string
	Key     string
	Payload string
}

// Row 是各服务共用的发件箱表
type Row struct {
	ID        string `gorm:"type:char(36);primaryKey"`
	Topic     string `gorm:"size:128;index"`
	BizKey    string `gorm:"column:biz_key;size:128"`
	Payload   string `gorm:"type:text"`
	Delivered bool   `gorm:"index"`
	CreatedAt time.Time
}

func (Row) TableName() string { return "outbox_events" }

// Insert 在当前事务里写入一条未投递事件
func Insert(tx *gorm.DB, topic, key, payload string) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return tx.Create(&Row{
		ID: id.String(), Topic: topic, BizKey: key, Payload: payload, Delivered: false,
	}).Error
}

// Store 供投递循环读取未投递事件
type Store interface {
	Pending(ctx context.Context, limit int) ([]Event, error)
	MarkDelivered(ctx context.Context, ids []string) error
}

type gormStore struct{ db *gorm.DB }

func NewStore(db *gorm.DB) Store { return &gormStore{db: db} }

func (s *gormStore) Pending(ctx context.Context, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []Row
	err := s.db.WithContext(ctx).Where("delivered = ?", false).Order("created_at").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, Event{ID: row.ID, Topic: row.Topic, Key: row.BizKey, Payload: row.Payload})
	}
	return out, nil
}

func (s *gormStore) MarkDelivered(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Model(&Row{}).Where("id IN ?", ids).Update("delivered", true).Error
}

// StartPublisher 每秒投递一批 连不上只记日志并稍后重试
func StartPublisher(store Store) func() {
	ctx, cancel := context.WithCancel(context.Background())
	go publishLoop(ctx, store)
	return cancel
}

func publishLoop(ctx context.Context, store Store) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var client *kgo.Client
	closeClient := func() {
		if client != nil {
			client.Close()
			client = nil
		}
	}
	defer closeClient()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if client == nil {
				cl, err := kgo.NewClient(
					kgo.SeedBrokers(Brokers()...),
					kgo.AllowAutoTopicCreation(),
					kgo.DialTimeout(2*time.Second),
					kgo.ProduceRequestTimeout(3*time.Second),
					kgo.RecordRetries(0),
					kgo.RecordDeliveryTimeout(4*time.Second),
				)
				if err != nil {
					log.Error("kafka client", "err", err)
					continue
				}
				client = cl
			}
			events, err := store.Pending(ctx, 100)
			if err != nil {
				log.Error("outbox pending", "err", err)
				continue
			}
			if len(events) == 0 {
				continue
			}
			recs := make([]*kgo.Record, 0, len(events))
			ids := make([]string, 0, len(events))
			for _, ev := range events {
				recs = append(recs, &kgo.Record{
					Topic:   ev.Topic,
					Key:     []byte(ev.Key),
					Value:   []byte(ev.Payload),
					Headers: []kgo.RecordHeader{{Key: "event_id", Value: []byte(ev.ID)}},
				})
				ids = append(ids, ev.ID)
			}
			pctx, pcancel := context.WithTimeout(ctx, 5*time.Second)
			err = client.ProduceSync(pctx, recs...).FirstErr()
			pcancel()
			if err != nil {
				log.Error("kafka produce", "err", err)
				closeClient()
				continue
			}
			if err := store.MarkDelivered(ctx, ids); err != nil {
				log.Error("outbox mark delivered", "err", err)
			}
		}
	}
}

// Apply 处理一条已拉取的事件 返回错误则原地退避重试
type Apply func(ctx context.Context, ev Event) error

// maxAttempts 是同一条记录最多尝试的次数 用完仍失败就记日志跳过
const maxAttempts = 5

// retryDelay 是第 n 次失败后的等待 从 200ms 起翻倍 封顶 3s
func retryDelay(n int) time.Duration {
	if n < 1 {
		n = 1
	}
	if n > 5 {
		return 3 * time.Second
	}
	return min(200*time.Millisecond<<(n-1), 3*time.Second)
}

// StartConsumer 拉取主题 每条原地重试 同一条连续失败 5 次后跳过
func StartConsumer(group string, topics []string, apply Apply) func() {
	ctx, cancel := context.WithCancel(context.Background())
	go consumeLoop(ctx, group, topics, apply)
	return cancel
}

func consumeLoop(ctx context.Context, group string, topics []string, apply Apply) {
	var client *kgo.Client
	defer func() {
		if client != nil {
			client.Close()
		}
	}()
	for {
		if ctx.Err() != nil {
			return
		}
		if client == nil {
			cl, err := kgo.NewClient(
				kgo.SeedBrokers(Brokers()...),
				kgo.ConsumerGroup(group),
				kgo.ConsumeTopics(topics...),
				kgo.DisableAutoCommit(),
				kgo.DialTimeout(2*time.Second),
			)
			if err != nil {
				log.Error("kafka consumer", "err", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Second):
				}
				continue
			}
			client = cl
		}
		pollCtx, pollCancel := context.WithTimeout(ctx, 2*time.Second)
		fetches := client.PollFetches(pollCtx)
		pollCancel()
		if ctx.Err() != nil {
			return
		}
		connErr := false
		for _, ferr := range fetches.Errors() {
			if ferr.Err == context.DeadlineExceeded || ferr.Err == context.Canceled {
				continue
			}
			connErr = true
			log.Error("kafka fetch", "topic", ferr.Topic, "err", ferr.Err)
		}
		var pending []*kgo.Record
		fetches.EachRecord(func(rec *kgo.Record) {
			pending = append(pending, rec)
		})
		consumeBatch(ctx, pending, apply, retryDelay, func(done []*kgo.Record) {
			// 停机时也要把已处理的记录提交掉 所以不跟随 ctx 取消
			cctx, ccancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
			defer ccancel()
			if err := client.CommitRecords(cctx, done...); err != nil {
				log.Error("kafka commit", "err", err)
			}
		})
		if connErr {
			client.Close()
			client = nil
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}
}

func eventFrom(rec *kgo.Record) Event {
	ev := Event{Topic: rec.Topic, Key: string(rec.Key), Payload: string(rec.Value)}
	for _, header := range rec.Headers {
		if header.Key == "event_id" {
			ev.ID = string(header.Value)
		}
	}
	// 没有事件头时用分区位置当 id 同一业务键的不同事件不能被当成重复
	if ev.ID == "" {
		ev.ID = fmt.Sprintf("%s/%d/%d", rec.Topic, rec.Partition, rec.Offset)
	}
	return ev
}

// consumeBatch 按分区逐条处理 每处理完一个分区就把连续成功或被跳过的前缀交给 commit
// 拉取位置已经前进 不提交也不会重拉 所以失败只能在这里原地重试
func consumeBatch(ctx context.Context, recs []*kgo.Record, apply Apply, delay func(int) time.Duration, commit func([]*kgo.Record)) {
	type part struct {
		topic string
		id    int32
	}
	groups := map[part][]*kgo.Record{}
	order := make([]part, 0)
	for _, rec := range recs {
		key := part{topic: rec.Topic, id: rec.Partition}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], rec)
	}
	for _, key := range order {
		partRecs := groups[key]
		sort.SliceStable(partRecs, func(i, j int) bool { return partRecs[i].Offset < partRecs[j].Offset })
		done := consumePartition(ctx, partRecs, apply, delay)
		if len(done) > 0 {
			commit(done)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// consumePartition 按偏移逐条处理 某条还没成功也没用完次数时不碰后面的记录 只有停机才会中途返回
func consumePartition(ctx context.Context, recs []*kgo.Record, apply Apply, delay func(int) time.Duration) []*kgo.Record {
	done := make([]*kgo.Record, 0, len(recs))
	for _, rec := range recs {
		if !applyWithRetry(ctx, eventFrom(rec), apply, delay) {
			break
		}
		done = append(done, rec)
	}
	return done
}

// applyWithRetry 成功或连续失败满 maxAttempts 次后返回 true 等待重试时停机返回 false
func applyWithRetry(ctx context.Context, ev Event, apply Apply, delay func(int) time.Duration) bool {
	for attempt := 1; ; attempt++ {
		if ctx.Err() != nil {
			return false
		}
		err := apply(ctx, ev)
		if err == nil {
			return true
		}
		log.Error("consume event", "id", ev.ID, "topic", ev.Topic, "err", err, "attempt", attempt)
		if attempt >= maxAttempts {
			log.Error("skip event after retries", "id", ev.ID, "topic", ev.Topic, "attempts", attempt)
			return true
		}
		timer := time.NewTimer(delay(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
	}
}
