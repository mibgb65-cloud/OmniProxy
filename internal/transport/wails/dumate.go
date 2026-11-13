package wails

import (
	"context"

	"omniproxy/internal/application/dumate"
)

// dumateService 复用账号服务构造 DuMate 应用服务；控制台客户端按默认 HTTPS 端点。
func (s *AppService) dumateService() (*dumate.Service, error) {
	accounts, err := s.accountService()
	if err != nil {
		return nil, err
	}
	return dumate.New(accounts, nil), nil
}

// DuMateImportDesktop 只读导入本机搭子桌面端登录态，并核对远端身份。
func (s *AppService) DuMateImportDesktop(providerID string) (dumate.ImportResult, error) {
	service, err := s.dumateService()
	if err != nil {
		return dumate.ImportResult{}, err
	}
	result, err := service.ImportDesktop(context.Background(), providerID)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return result, err
}

// DuMateImportCookie 导入用户粘贴的 Cookie 请求头，不会把 Cookie 返回前端。
func (s *AppService) DuMateImportCookie(providerID, cookie string) (dumate.ImportResult, error) {
	service, err := s.dumateService()
	if err != nil {
		return dumate.ImportResult{}, err
	}
	result, err := service.ImportCookie(context.Background(), providerID, cookie)
	if err == nil {
		s.refreshAgentCatalog()
	}
	return result, err
}

// DuMatePoints 查询公开积分与签到状态；查询失败用 State／ErrorCode 表达。
func (s *AppService) DuMatePoints(accountID string) (dumate.Points, error) {
	service, err := s.dumateService()
	if err != nil {
		return dumate.Points{}, err
	}
	return service.Points(context.Background(), accountID)
}

// DuMateCheckin 手动签到；结果区分已领取、今日已完成、待到账与失败。
func (s *AppService) DuMateCheckin(accountID string) (dumate.Checkin, error) {
	service, err := s.dumateService()
	if err != nil {
		return dumate.Checkin{}, err
	}
	return service.Checkin(context.Background(), accountID)
}
