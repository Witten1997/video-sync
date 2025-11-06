# 如何获取 B 站 Cookies 配置指南

## 为什么需要 Cookies？

B 站的部分 API（如视频详细信息、弹幕等）需要登录凭证才能访问。通过配置 cookies，bili-download 可以模拟登录状态来获取这些信息。

## 获取 Cookies 的方法

### 方法 1：使用浏览器开发者工具（推荐）

#### Chrome/Edge 浏览器

1. **登录 B 站**
   - 打开浏览器，访问 https://www.bilibili.com
   - 使用你的账号登录

2. **打开开发者工具**
   - 按 `F12` 键，或右键点击页面选择"检查"
   - 切换到 `Application` 标签（或 `应用程序`）

3. **查看 Cookies**
   - 在左侧面板展开 `Storage` → `Cookies`
   - 点击 `https://www.bilibili.com`

4. **复制需要的 Cookie 值**
   找到并复制以下 Cookie 的值：

   | Cookie 名称 | 必需 | 说明 |
   |------------|------|------|
   | `SESSDATA` | ✅ 必需 | 会话令牌，最重要的凭证 |
   | `bili_jct` | ✅ 必需 | CSRF 令牌 |
   | `buvid3` | 推荐 | 设备标识 |
   | `DedeUserID` | 可选 | 用户 ID |
   | `ac_time_value` | 可选 | 访问时间值 |

   ![Cookie 获取示例](https://i.imgur.com/example.png)

#### Firefox 浏览器

1. **登录 B 站**
   - 访问 https://www.bilibili.com 并登录

2. **打开开发者工具**
   - 按 `F12` 键
   - 切换到 `存储` 标签

3. **查看 Cookies**
   - 展开 `Cookie` → `https://www.bilibili.com`
   - 找到并复制上述 Cookie 值

### 方法 2：使用浏览器扩展（简单快捷）

#### Get cookies.txt LOCALLY 扩展

1. **安装扩展**
   - Chrome: https://chrome.google.com/webstore/detail/get-cookiestxt-locally/
   - Firefox: https://addons.mozilla.org/firefox/addon/get-cookies-txt-locally/

2. **获取 Cookies**
   - 登录 B 站后，点击扩展图标
   - 点击 "Export" 导出 cookies
   - 从导出的内容中找到需要的值

### 方法 3：使用 Python 脚本（自动化）

```python
#!/usr/bin/env python3
"""
自动获取 B 站 Cookies 的脚本
需要安装: pip install selenium
"""

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
import time

def get_bilibili_cookies():
    # 设置浏览器选项
    options = Options()
    # options.add_argument('--headless')  # 无头模式（可选）

    # 启动浏览器
    driver = webdriver.Chrome(options=options)

    try:
        # 访问 B 站
        driver.get('https://www.bilibili.com')

        print("请在浏览器中登录 B 站...")
        print("登录完成后按 Enter 键继续...")
        input()

        # 获取所有 cookies
        cookies = driver.get_cookies()

        # 提取需要的 cookies
        result = {}
        target_cookies = ['SESSDATA', 'bili_jct', 'buvid3', 'DedeUserID', 'ac_time_value']

        for cookie in cookies:
            if cookie['name'] in target_cookies:
                result[cookie['name']] = cookie['value']

        # 输出配置格式
        print("\n=== 配置信息 ===")
        print("bilibili:")
        print("  credential:")
        print(f"    sessdata: \"{result.get('SESSDATA', '')}\"")
        print(f"    bili_jct: \"{result.get('bili_jct', '')}\"")
        print(f"    buvid3: \"{result.get('buvid3', '')}\"")
        print(f"    dedeuserid: \"{result.get('DedeUserID', '')}\"")
        print(f"    ac_time_value: \"{result.get('ac_time_value', '')}\"")

        return result

    finally:
        driver.quit()

if __name__ == '__main__':
    get_bilibili_cookies()
```

## 配置到 bili-download

### 1. 找到配置文件

配置文件位置：`configs/config.yaml`

如果没有，复制示例配置：
```bash
cp configs/config.example.yaml configs/config.yaml
```

### 2. 编辑配置文件

打开 `configs/config.yaml`，找到 `bilibili.credential` 部分：

```yaml
# B站认证
bilibili:
  credential:
    sessdata: "你的_SESSDATA_值"
    bili_jct: "你的_bili_jct_值"
    buvid3: "你的_buvid3_值"
    dedeuserid: "你的_DedeUserID_值"
    ac_time_value: "你的_ac_time_value_值"
```

### 3. 填入 Cookie 值

**示例**：
```yaml
bilibili:
  credential:
    sessdata: "1a2b3c4d%2C1234567890%2Cabcde*12"
    bili_jct: "1234567890abcdef1234567890abcdef"
    buvid3: "ABCD1234-EF56-7890-ABCD-1234567890AB"
    dedeuserid: "123456789"
    ac_time_value: "1234567890abcdef"
```

**注意事项**：
- ✅ 保持引号
- ✅ 不要有多余的空格
- ✅ `SESSDATA` 和 `bili_jct` 是必需的
- ⚠️ Cookies 敏感信息，不要泄露给他人

### 4. 重启服务

配置完成后，重启 bili-download 服务：

```bash
# 停止当前服务（Ctrl+C）
# 重新启动
./build/bili-download.exe
```

## 验证配置是否成功

### 方法 1：查看日志

启动服务后，如果配置正确，日志会显示：
```
✓ B 站认证信息已配置
```

如果配置错误，会显示：
```
⚠ B 站认证信息未配置或无效
```

### 方法 2：测试下载

尝试通过 URL 下载一个视频：
```bash
curl -X POST http://localhost:8080/api/videos/download-by-url \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{"url":"https://www.bilibili.com/video/BV1xx411c7XD"}'
```

如果成功，会返回：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "xxx",
    "video": {...}
  }
}
```

## Cookie 有效期和刷新

### Cookie 过期时间

- `SESSDATA`: 通常有效期 **6 个月**
- 其他 Cookie: 有效期不一

### Cookie 过期后怎么办？

如果遇到以下错误：
```
获取视频信息失败: 账号未登录
```

说明 Cookie 已过期，需要重新获取：
1. 在浏览器中重新登录 B 站
2. 按照上述步骤重新获取 Cookie
3. 更新配置文件
4. 重启服务

### 自动刷新（可选）

可以编写脚本定期检查和更新 Cookie：

```bash
#!/bin/bash
# auto-refresh-cookies.sh

while true; do
  # 检查 API 是否返回未登录错误
  response=$(curl -s http://localhost:8080/api/system/info)

  if echo "$response" | grep -q "账号未登录"; then
    echo "检测到 Cookie 过期，请重新获取"
    # 发送通知...
  fi

  # 每天检查一次
  sleep 86400
done
```

## 安全建议

### 1. 保护你的 Cookies

- ❌ 不要在公共场合展示配置文件
- ❌ 不要将 cookies 提交到 Git 仓库
- ✅ 使用 `.gitignore` 忽略配置文件
- ✅ 使用环境变量存储敏感信息

### 2. 使用环境变量（可选）

除了配置文件，还可以使用环境变量：

```bash
export BILI_SESSDATA="your_sessdata"
export BILI_BILI_JCT="your_bili_jct"
export BILI_BUVID3="your_buvid3"

./build/bili-download.exe
```

### 3. 定期更新

建议每 3-6 个月更新一次 Cookie，即使未过期。

### 4. 使用小号（推荐）

- 建议专门注册一个小号用于下载
- 避免使用主账号，防止被风控

## 常见问题

### Q1: SESSDATA 在哪里？

A: 在浏览器开发者工具的 Application → Cookies → https://www.bilibili.com 中查找。

### Q2: Cookie 值很长，复制不完整怎么办？

A:
- 双击 Cookie 值单元格
- 使用 Ctrl+C 复制
- 或者点击 Cookie 查看完整值

### Q3: 配置后还是提示"账号未登录"？

A: 检查以下几点：
1. ✅ SESSDATA 和 bili_jct 是否都已配置
2. ✅ 值是否复制完整（没有截断）
3. ✅ 引号是否正确
4. ✅ 是否重启了服务
5. ✅ Cookie 是否已过期

### Q4: 会被 B 站封号吗？

A:
- 正常使用不会被封号
- 避免频繁请求（已有限流保护）
- 建议使用小号

### Q5: 可以使用别人的 Cookie 吗？

A:
- ❌ 不建议，这是账号凭证
- ⚠️ 可能导致账号安全问题
- ⚠️ 对方可以使用你的账号

## 进阶：Cookie 池（多账号）

如果需要高频使用，可以配置多个账号的 Cookie 轮流使用：

```yaml
bilibili:
  credentials:
    - sessdata: "account1_sessdata"
      bili_jct: "account1_bili_jct"
    - sessdata: "account2_sessdata"
      bili_jct: "account2_bili_jct"

  # 请求策略
  strategy: "round-robin"  # 轮询
  # strategy: "random"     # 随机
```

## 技术原理

### WBI 签名机制

B 站使用 WBI（Web Bilibili Interface）签名来保护 API：

1. 获取 WBI 图片 URL（需要登录）
2. 从 URL 中提取密钥
3. 对请求参数进行排序和编码
4. 计算 MD5 签名
5. 添加签名到请求

没有有效的登录凭证，无法获取 WBI 图片 URL，因此无法完成签名。

### Cookie 的作用

```
SESSDATA     → 会话标识，证明你已登录
bili_jct     → CSRF 令牌，防止跨站请求伪造
buvid3       → 设备标识
DedeUserID   → 用户 ID
ac_time_value → 访问时间值（防刷）
```

## 参考资料

- [B 站 API 文档](https://github.com/SocialSisterYi/bilibili-API-collect)
- [WBI 签名算法](https://github.com/SocialSisterYi/bilibili-API-collect/blob/master/docs/misc/sign/wbi.md)
- [Cookie 安全最佳��践](https://owasp.org/www-community/controls/SecureCookieAttribute)
