package account

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
