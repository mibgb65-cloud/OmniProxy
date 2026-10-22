// Package modeleffort 保存每个模型固定的思考档位；网关在请求侧按它覆盖客户端选择。
//
// 只保存取值，不推断能力：写入方必须先按模型已确认的推理档位枚举校验，
// 未确认档位的模型不会出现在任何写入路径里。
package modeleffort

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// Pins 返回全部固定档位，键是模型 ID。
func Pins(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT model_id, effort FROM model_effort_pins ORDER BY model_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pins := map[string]string{}
	for rows.Next() {
		var model, effort string
		if err := rows.Scan(&model, &effort); err != nil {
			return nil, err
		}
		pins[model] = effort
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return pins, nil
}

// Save 在同一事务里写入固定档位；空档位删除该模型的固定值。
func Save(ctx context.Context, db execer, pins map[string]string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	for model, effort := range pins {
		model = strings.TrimSpace(model)
		effort = strings.TrimSpace(effort)
		if model == "" {
			continue
		}
		if effort == "" {
			if _, err := db.ExecContext(ctx, `DELETE FROM model_effort_pins WHERE model_id=?`, model); err != nil {
				return err
			}
			continue
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO model_effort_pins(model_id,effort,updated_at) VALUES(?,?,?)
			ON CONFLICT(model_id) DO UPDATE SET effort=excluded.effort, updated_at=excluded.updated_at`,
			model, effort, now); err != nil {
			return err
		}
	}
	return nil
}
