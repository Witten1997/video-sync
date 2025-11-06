# B站认证信息验证功能

## 功能说明

在配置页面的"B站认证"标签中，当点击"保存配置"按钮后，系统会自动验证配置的认证信息是否有效。

## 实现细节

### 后端实现

#### 1. API 端点

新增了验证认证信息的 API 端点：

```
POST /api/config/validate-credential
```

#### 2. 验证逻辑 (internal/bilibili/client.go)

```go
func (c *Client) ValidateCredential() error {
    // 1. 使用 CheckCredentialValid 方法检查 Cookie 是否需要刷新
    valid, err := c.CheckCredentialValid()
    if err != nil {
        return fmt.Errorf("验证失败: %w", err)
    }

    if !valid {
        return fmt.Errorf("账号未登录或 Cookie 已过期")
    }

    // 2. 尝试获取用户信息确认登录状态
    _, err = c.GetMe()
    if err != nil {
        return fmt.Errorf("获取用户信息失败: %w", err)
    }

    return nil
}
```

验证过程：
1. 调用 B站的 Cookie 信息接口检查凭证有效性
2. 调用用户信息接口获取登录用户的详细信息
3. 如果两步都成功，则认证有效

#### 3. API 处理器 (internal/api/handler_config.go)

```go
func (s *Server) handleValidateBilibiliCredential(c *gin.Context) {
    // 验证当前配置的B站凭证
    if err := s.biliClient.ValidateCredential(); err != nil {
        respondError(c, 400, fmt.Sprintf("认证验证失败: %v", err))
        return
    }

    // 获取用户信息
    userInfo, err := s.biliClient.GetMe()
    if err != nil {
        respondError(c, 500, fmt.Sprintf("获取用户信息失败: %v", err))
        return
    }

    respondSuccess(c, gin.H{
        "valid":     true,
        "message":   "认证信息有效",
        "user_info": userInfo,
    })
}
```

响应格式：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "valid": true,
    "message": "认证信息有效",
    "user_info": {
      "mid": 123456789,
      "uname": "用户名",
      "face": "https://...",
      "sign": "个性签名",
      "level": 6,
      "vip_type": 2,
      "vip_status": 1
    }
  }
}
```

### 前端实现

#### 1. API 调用 (web/src/api/config.ts)

```typescript
export const validateBilibiliCredential = () => {
  return http.post<{
    valid: boolean
    message: string
    user_info?: {
      mid: number
      uname: string
      face: string
      sign: string
      level: number
      vip_type: number
      vip_status: number
    }
  }>('/config/validate-credential')
}
```

#### 2. 配置页面 (web/src/views/Config.vue)

##### 验证状态显示

在 B站认证表单中添加了验证状态显示区域：

```vue
<el-form-item v-if="credentialValidation.show" label="验证状态">
  <el-alert
    :type="credentialValidation.valid ? 'success' : 'error'"
    :title="credentialValidation.message"
    :closable="false"
    show-icon
  >
    <template v-if="credentialValidation.valid && credentialValidation.userInfo" #default>
      <div class="user-info">
        <p><strong>用户名:</strong> {{ credentialValidation.userInfo.uname }}</p>
        <p><strong>UID:</strong> {{ credentialValidation.userInfo.mid }}</p>
        <p><strong>等级:</strong> Lv{{ credentialValidation.userInfo.level }}</p>
        <p v-if="credentialValidation.userInfo.vip_status === 1">
          <strong>会员状态:</strong> 大会员
        </p>
      </div>
    </template>
  </el-alert>
</el-form-item>
```

##### 自动验证逻辑

```typescript
// 保存配置
const handleSave = async () => {
  loading.value = true
  try {
    const configToSubmit = getCurrentTabConfig()
    await updateConfig(configToSubmit)
    ElMessage.success('保存成功')

    // 如果保存的是B站认证配置，自动验证
    if (activeTab.value === 'bilibili') {
      await validateCredential()
    }
  } catch (error) {
    console.error('保存配置失败:', error)
  } finally {
    loading.value = false
  }
}

// 验证B站认证信息
const validateCredential = async () => {
  credentialValidation.value.show = false

  try {
    const result = await validateBilibiliCredential()

    credentialValidation.value = {
      show: true,
      valid: true,
      message: result.message || '认证信息有效',
      userInfo: result.user_info
    }

    ElMessage.success('认证信息验证成功')
  } catch (error: any) {
    const errorMsg = error?.response?.data?.message || error?.message || '认证验证失败'

    credentialValidation.value = {
      show: true,
      valid: false,
      message: errorMsg,
      userInfo: undefined
    }

    ElMessage.error(errorMsg)
  }
}
```

## 使用流程

### 1. 配置认证信息

1. 打开配置页面
2. 切换到"B站认证"标签
3. 填入从浏览器获取的 Cookie 信息：
   - SESSDATA（必需）
   - bili_jct（必需）
   - buvid3（推荐）
   - DedeUserID（可选）
   - ac_time_value（可选）

### 2. 保存并验证

1. 点击"保存配置"按钮
2. 系统自动保存配置到文件
3. 自动调用验证接口检查认证信息
4. 显示验证结果

### 3. 查看验证结果

#### 验证成功

显示绿色的成功提示框，包含：
- ✅ 认证信息有效
- 用户名
- UID
- 等级
- 会员状态（如果是大会员）

#### 验证失败

显示红色的错误提示框，包含：
- ❌ 错误原因
  - "账号未登录或 Cookie 已过期"
  - "获取用户信息失败: ..."
  - 其他网络或API错误

## 常见错误及解决方法

### 1. "账号未登录或 Cookie 已过期"

**原因：**
- SESSDATA 或 bili_jct 不正确
- Cookie 已过期（通常6个月有效期）
- Cookie 值复制不完整

**解决方法：**
1. 重新登录 B 站
2. 重新获取 Cookie
3. 确保复制完整的 Cookie 值
4. 检查是否有多余的空格或引号

### 2. "获取用户信息失败"

**原因：**
- 网络连接问题
- B 站 API 临时不可用
- Cookie 格式错误

**解决方法：**
1. 检查网络连接
2. 稍后重试
3. 检查 Cookie 格式是否正确

### 3. "验证失败: 请求失败"

**原因：**
- 无法连接到 B 站服务器
- 防火墙或代理问题

**解决方法：**
1. 检查网络连接
2. 检查防火墙设置
3. 如果使用代理，检查代理配置

## 技术细节

### 验证流程

```
用户点击保存
    ↓
保存配置到文件 (config.yaml)
    ↓
更新内存中的配置 (s.config)
    ↓
重新创建 biliClient（使用新的 credential）
    ↓
调用验证接口 (/api/config/validate-credential)
    ↓
后端调用 B 站 API
    ├─→ /x/passport-login/web/cookie/info (检查 Cookie)
    └─→ /x/web-interface/nav (获取用户信息)
    ↓
返回验证结果和用户信息
    ↓
前端显示结果
```

### 相关文件

**后端：**
- `internal/bilibili/client.go` - ValidateCredential() 方法
- `internal/bilibili/user.go` - GetMe(), CheckCredentialValid() 方法
- `internal/api/handler_config.go` - handleValidateBilibiliCredential() 处理器
- `internal/api/server.go` - 路由注册

**前端：**
- `web/src/api/config.ts` - validateBilibiliCredential() API 函数
- `web/src/views/Config.vue` - UI 和验证逻辑

## 安全说明

1. **Cookie 存储**
   - Cookie 以明文形式存储在 `configs/config.yaml`
   - 建议不要将配置文件提交到版本控制系统
   - `.gitignore` 应包含 `configs/config.yaml`

2. **传输安全**
   - 前后端通信建议使用 HTTPS
   - 可以通过 AuthToken 鉴权保护 API

3. **使用小号**
   - 建议使用专门的小号而不是主账号
   - 避免使用大会员账号

## 未来改进

1. **Cookie 加密存储**
   - 使用加密算法加密存储 Cookie
   - 使用密钥管理工具

2. **自动刷新**
   - Cookie 快过期时自动提醒
   - 支持自动刷新机制

3. **多账号支持**
   - 支持配置多个账号的 Cookie
   - 自动轮询使用不同账号

4. **更详细的验证信息**
   - 显示 Cookie 过期时间
   - 显示账号风控状态
   - 显示限流情况

## 参考文档

- [如何获取 B 站 Cookies](./HOW_TO_GET_COOKIES.md)
- [通过 URL 下载视频使用指南](./USAGE_URL_DOWNLOAD.md)
- [B 站 API 文档](https://github.com/SocialSisterYi/bilibili-API-collect)
