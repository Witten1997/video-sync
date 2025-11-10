package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"bili-download/internal/bilibili"
	"bili-download/internal/config"
	"bili-download/internal/utils"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// handleQRCodeGenerate 生成二维码
func (s *Server) handleQRCodeGenerate(c *gin.Context) {
	// 创建二维码登录管理器
	qrLogin := bilibili.NewQRLogin()

	// 申请二维码（Web 端）
	resp, err := qrLogin.GenerateQRCode()
	if err != nil {
		utils.Error("生成二维码失败: %v", err)
		respondError(c, http.StatusInternalServerError, "生成二维码失败: "+err.Error())
		return
	}

	// 返回二维码信息
	respondSuccess(c, gin.H{
		"url":        resp.Data.URL,       // 二维码内容 URL
		"qrcode_key": resp.Data.QRCodeKey, // 扫码登录秘钥
		"expires_in": 180,                 // 超时时间（秒）
	})
}

// handleQRCodePoll 轮询二维码状态
func (s *Server) handleQRCodePoll(c *gin.Context) {
	qrcodeKey := c.Query("qrcode_key")
	if qrcodeKey == "" {
		// 向后兼容 auth_code 参数名
		qrcodeKey = c.Query("auth_code")
	}
	if qrcodeKey == "" {
		respondError(c, http.StatusBadRequest, "缺少参数: qrcode_key")
		return
	}

	// 创建二维码登录管理器
	qrLogin := bilibili.NewQRLogin()

	// 轮询二维码状态（Web 端）
	result, err := qrLogin.PollQRCode(qrcodeKey)
	if err != nil {
		utils.Error("轮询二维码状态失败: %v", err)
		respondError(c, http.StatusInternalServerError, "轮询二维码状态失败: "+err.Error())
		return
	}

	// 如果登录成功，保存凭据到配置文件
	if result.Status == bilibili.QRCodeStatusSuccess && result.Credential != nil {
		if err := s.saveCredentialToConfig(result.Credential); err != nil {
			utils.Error("保存登录凭据失败: %v", err)
			respondError(c, http.StatusInternalServerError, "保存登录凭据失败: "+err.Error())
			return
		}

		// 更新内存中的凭据
		s.biliClient.SetCredential(result.Credential)

		utils.Info("B站二维码登录成功，凭据已保存")
	}

	// 返回状态信息
	respondSuccess(c, gin.H{
		"status":  result.Status,
		"message": result.Message,
	})
}

// saveCredentialToConfig 保存凭据到配置文件
func (s *Server) saveCredentialToConfig(credential *bilibili.Credential) error {
	// 确定配置文件路径
	configPath := s.configPath
	if configPath == "" {
		configPath = "./configs/config.yaml"
	}

	// 读取现有配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析 YAML
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(data, &configMap); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 更新 bilibili.credential 部分
	if configMap["bilibili"] == nil {
		configMap["bilibili"] = make(map[string]interface{})
	}

	biliConfig, ok := configMap["bilibili"].(map[string]interface{})
	if !ok {
		biliConfig = make(map[string]interface{})
		configMap["bilibili"] = biliConfig
	}

	// 设置凭据
	biliConfig["credential"] = map[string]string{
		"sessdata":      credential.SESSDATA,
		"bili_jct":      credential.BiliJct,
		"buvid3":        credential.Buvid3,
		"dedeuserid":    credential.DedeUserID,
		"ac_time_value": credential.AcTimeValue,
	}

	// 序列化为 YAML
	updatedData, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 创建配置目录（如果不存在）
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 写回配置文件
	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	// 重新加载配置到内存
	newConfig, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("重新加载配置失败: %w", err)
	}

	// 更新服务器的配置引用
	s.config = newConfig

	return nil
}
