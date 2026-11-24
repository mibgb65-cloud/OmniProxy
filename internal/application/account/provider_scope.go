package account

import (
	"context"

	"omniproxy/internal/core/netproxy"
)

// OutboundContext 把账号所属提供商写进出站上下文。
//
// 推理请求由网关写入提供商作用域；额度、余额、目录与授权续期也必须带上同一身份，
// 否则「只给这个提供商配置了代理」会被主机名回退绕过而悄悄直连。
func (i Item) OutboundContext(ctx context.Context) context.Context {
	return netproxy.WithProvider(ctx, i.ProviderID, "")
}

// ProviderScope 返回账号选项决定的地区与计费方式。
// 网关记账身份与用量查询必须使用同一取值，否则同一账号的历史会被拆到不同主体下。
func (i Item) ProviderScope() (region, billingMode string) {
	if i.StepFun != nil {
		region, billingMode = i.StepFun.Region, i.StepFun.BillingMode
	}
	if i.MiniMax != nil {
		region, billingMode = i.MiniMax.Region, i.MiniMax.BillingMode
	}
	if i.MiMo != nil {
		region, billingMode = i.MiMo.Region, i.MiMo.BillingMode
	}
	if i.Kimi != nil {
		region, billingMode = i.Kimi.Region, i.Kimi.Mode()
	}
	return region, billingMode
}
