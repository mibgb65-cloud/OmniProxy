package account

import (
	"fmt"
	"time"
)

func nowAfter(previous string) (string, error) {
	return timestampAfter(previous, time.Now())
}

// updated_at 同时是乐观锁版本，不能假设墙上时钟每次调用都前进。
// 下界来自数据库旧值，配合 WHERE updated_at 校验，可覆盖重启、并发实例及回拨；
// 不使用进程内计数器，也不改变已有 RFC3339Nano 协议和数据库结构。
func timestampAfter(previous string, current time.Time) (string, error) {
	if before, err := time.Parse(time.RFC3339Nano, previous); err == nil && !current.After(before) {
		current = before.Add(time.Nanosecond)
	}
	// 历史非时间格式值可归一化；但新值必须可往返解析，极值溢出不能写入账号或凭据。
	current = current.UTC()
	if current.Year() < 0 || current.Year() > 9999 {
		return "", fmt.Errorf("%w: timestamp out of range", ErrInvalidInput)
	}
	return current.Format(time.RFC3339Nano), nil
}
