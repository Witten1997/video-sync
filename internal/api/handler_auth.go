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
	qrLogin := bilibili.NewQRLogin(s.config)

	resp, err := qrLogin.GenerateQRCode()
	if err != nil {
		utils.Error("生成二维码失败: %v", err)
		respondError(c, http.StatusInternalServerError, "生成二维码失败: "+err.Error())
		return
	}

	respondSuccess(c, gin.H{
		"url":        resp.Data.URL,
		"qrcode_key": resp.Data.QRCodeKey,
		"expires_in": 180,
	})
}

// handleQRCodePoll 轮询二维码状态
func (s *Server) handleQRCodePoll(c *gin.Context) {
	qrcodeKey := c.Query("qrcode_key")
	if qrcodeKey == "" {
		qrcodeKey = c.Query("auth_code")
	}
	if qrcodeKey == "" {
		respondError(c, http.StatusBadRequest, "缺少参数: qrcode_key")
		return
	}

	qrLogin := bilibili.NewQRLogin(s.config)

	result, err := qrLogin.PollQRCode(qrcodeKey)
	if err != nil {
		utils.Error("轮询二维码状态失败: %v", err)
		respondError(c, http.StatusInternalServerError, "轮询二维码状态失败: "+err.Error())
		return
	}

	if result.Status == bilibili.QRCodeStatusSuccess && result.Credential != nil {
		if err := s.saveCredentialToConfig(result.Credential); err != nil {
			utils.Error("保存登录凭据失败: %v", err)
			respondError(c, http.StatusInternalServerError, "保存登录凭据失败: "+err.Error())
			return
		}

		s.biliClient.UpdateConfig(s.config)
		utils.Info("B站二维码登录成功，凭据已保存")
	}

	respondSuccess(c, gin.H{
		"status":  result.Status,
		"message": result.Message,
	})
}

// saveCredentialToConfig 保存凭据到配置文件
func (s *Server) saveCredentialToConfig(credential *bilibili.Credential) error {
	configPath := s.configPath
	if configPath == "" {
		configPath = "./configs/config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	var configMap map[string]interface{}
	if err := yaml.Unmarshal(data, &configMap); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	if configMap["bilibili"] == nil {
		configMap["bilibili"] = make(map[string]interface{})
	}

	biliConfig, ok := configMap["bilibili"].(map[string]interface{})
	if !ok {
		biliConfig = make(map[string]interface{})
		configMap["bilibili"] = biliConfig
	}

	biliConfig["credential"] = map[string]string{
		"sessdata":      credential.SESSDATA,
		"bili_jct":      credential.BiliJct,
		"buvid3":        credential.Buvid3,
		"dedeuserid":    credential.DedeUserID,
		"ac_time_value": credential.AcTimeValue,
	}

	updatedData, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	newConfig, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("重新加载配置失败: %w", err)
	}

	s.config = newConfig
	s.biliClient.UpdateConfig(newConfig)
	s.refreshHTTPClients()

	return nil
}
