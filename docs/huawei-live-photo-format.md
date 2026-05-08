# 华为动态照片（Live 图）格式分析

> 数据来源：
> 1. nova 12 Pro（`ADA-AL00`，EMUI/HarmonyOS 4.2）相机直拍 `IMG_20260507_225522.jpg` 字节级解剖。
> 2. 第三方 App「萌制作」导出的 live 图 `0537680e-…jpg` 反向工程。
> 3. video-sync 在小红书 live 图素材上多轮 A/B 验证（test_A ~ test_G），最终在 test_G 通过华为相册识别。

## 1. 文件总览（华为相机直拍样本）

```
[SOI FFD8]
[APP1 Exif]                        9964 B  ← 标准 EXIF/GPS
[APP2 ICC]                          596 B  ← Huawei sRGB→DCI-P3 profile
[APP7 'HUAWEI']                  32974 B  ← HwMnote 拍摄参数（场景/AI 等）
[APP8 'HUAWEI']                  55788 B  ← TIFF 容器, 标签命名空间 0x20xx, 首字段 ASCII = "awb_ext1"
[APP9 'HUAWEI']                  43684 B  ← TIFF 容器, 标签命名空间 0x40xx, 首字段 ASCII = "awb_ext2"
[APP10 'HUAWEI']                    36 B  ← 单标签 0xFFFF=0x00（哨兵）
[APP0 JFIF + DQT/DHT/SOF/SOS]
[压缩流]
[EOI FFD9]                                ← JPEG 至此结束（offset 1,905,641）
[padding]                          1208 B  ← 内容非全 0，但内容不参与校验
[ftyp 'mp42']                           ← 嵌入 MP4 起点（offset 1,906,853）
[完整 H.264 + AAC MP4]          1,835,353 B
[LIVE footer]                        40 B  ← 华为私有魔法尾（关键！）
EOF                                       ← total 3,742,242 B
```

## 2. 关键结论

### 2.1 真正决定识别的是「文件末尾 40 字节 LIVE footer」

早期假设「JPEG `FFD9 EOI` 之后能扫到合法 `ftyp`」是必要条件 —— 这是必要但**不充分**条件。

实测华为相册识别动态照片的硬性条件是：**文件最后 40 字节必须是固定格式的 LIVE footer**。
没有这 40 字节，即使 JPEG + MP4 结构完全合法、ftyp 是 `mp42`、moov.meta 写齐 `com.android.*` 元数据，相册依然只把它当普通 JPEG 显示，不会出现「动态照片」标识、不会播放视频。

### 2.2 LIVE footer 格式规范

固定 40 字节，全 ASCII，分两段，每段 20 字节，不足用空格 `0x20` 右补齐：

```
[0..19]   = "<W>:<H>" + spaces        // 例 "1024:542            "
[20..39]  = "LIVE_<N>" + spaces       // 例 "LIVE_850176         "
```

| 字段 | 含义 | 校验情况 |
|---|---|---|
| `W:H`   | 名义上的宽高 | **不校验**。华为真机文件视频实为 960×720，footer 却写 `766:533`；小红书素材实为 1080×1440，写 `1024:542` 也能识别 |
| `LIVE_<N>` | 嵌入 MP4 的字节数（不含 footer 自身） | **必须严格匹配实际追加 mp4 长度**，否则识别失败 |

字节级示例（test_G，可识别）：

```
hex:   31 30 32 34 3a 35 34 32 20 20 20 20 20 20 20 20 20 20 20 20
       4c 49 56 45 5f 38 35 30 31 37 36 20 20 20 20 20 20 20 20 20
ascii: '1024:542            LIVE_850176         '
```

### 2.3 EOI 与 ftyp 之间的 padding：内容随意，长度不参与校验

华为真机样本是 1208 B 非零 padding，社区也验证过 0 B padding 同样识别。
video-sync 实现选择固定 16 B 0 padding（来源于 test_G 已验证样本，纯保守对齐）。

### 2.4 三段华为自定义 APP 段（APP7/8/9）与 live 图无关

| 段 | 用途 |
|---|---|
| APP7 (HwMnote)                  | 拍摄场景/AI/人像参数 |
| APP8 (`awb_ext1`)               | 白平衡扩展校准查表 |
| APP9 (`awb_ext2`)               | 白平衡扩展校准查表 |
| APP10 (单标签 `0xFFFF=0`)       | 哨兵/版本占位 |

它们是镜头色彩的私有校准数据，**复刻 live 图时完全不需要**。

### 2.5 内嵌 MP4 元数据：`com.android.manufacturer="HUAWEI"` 不是必需的

早期假设这条 mdta 键是识别关键，实测不是 —— test_G 的 mp4 不写任何 `com.android.*` 元数据，只要 footer 正确就能识别。
mdta 元数据可能影响 EXIF 详情面板的「设备信息」展示，但不影响识别本身。

### 2.6 全文搜索：识别与公开规范无关

文件中搜 `MovingPhoto` `LivePhoto` `MotionPhoto` `MicroVideo` `GCamera` 等公开规范关键字，**全部 0 命中**。
华为相册既不读 XMP，也不读 GCamera 标准，**只看末尾 40 字节 LIVE footer**。

## 3. 合成最小可识别 live 图的要求

按重要性排列：

1. **必需**：文件末尾追加 40 字节 LIVE footer（格式见 2.2）。
2. **必需**：JPEG 完整且以 `FFD9 EOI` 正常结束。
3. **必需**：紧跟一段合法 MP4（任意有效 ftyp，`isom`/`mp42` 均可），含视频轨；音频轨建议有但非硬性。
4. **可选**：JPEG 与 MP4 之间填若干字节（0~1212 B 都验证过可识别）。
5. **可选**：MP4 内 `moov.meta(mdta).com.android.*` 元数据（影响详情展示，不影响识别）。
6. **可选**：XMP-GCamera/XMP-Camera 等元数据（华为不看，但 Pixel/小米/OPPO 仍需要）。

## 4. video-sync 当前实现

代码位置：`internal/xhs/livephoto.go` `CreateLivePhoto`

```
输出文件 = 标准化 JPEG          ← 经 ffmpeg 转 JPEG，强制写入 APP0 JFIF 段
        + 16 字节 0 padding    ← buildHuaweiLiveFooter 之前的固定 padding
        + 标准化 MP4           ← ffmpeg 重封装，必有音轨（华为对无音轨视频有时拒识）
        + 40 字节 LIVE footer  ← buildHuaweiLiveFooter(mp4Size)
```

footer 构造（`livephoto.go::buildHuaweiLiveFooter`）：

```go
func buildHuaweiLiveFooter(mp4Size int64) []byte {
    footer := bytes.Repeat([]byte{' '}, 40)
    copy(footer[:20], []byte("1024:542"))
    copy(footer[20:], []byte(fmt.Sprintf("LIVE_%d", mp4Size)))
    return footer
}
```

`W:H` 写死为 `1024:542`（已验证可识别值，不参与校验）。`LIVE_<N>` 中的 `N` 取追加 MP4 文件的字节数。

## 5. 验证方法

```bash
# 看尾部 40 字节 footer
python -c "f=open('out.jpg','rb');f.seek(-40,2);print(repr(f.read().decode('latin-1')))"
# 期望输出形如：'1024:542            LIVE_<mp4字节数>     '

# 看 EOI 到 ftyp 的 padding 长度
python -c "
import sys
d=open('out.jpg','rb').read()
ftyp=d.find(b'ftyp')-4
eoi=d.rfind(b'\xff\xd9',0,ftyp)
print('pad:', ftyp-(eoi+2))
print('mp4 len:', len(d)-40-ftyp)
"

# ftyp brand
ffprobe -v error -show_entries format=major_brand -of default=nw=1 out.jpg
```

期望：

```
footer: '1024:542            LIVE_<mp4字节数>     '
pad:    16
mp4 len 与 footer 中的 LIVE_<N> 严格一致
```

## 6. 历史曲线（test_A ~ test_G 验证路径）

| 测试 | 构造 | 华为识别 | 结论 |
|---|---|---|---|
| A | 我们的 JPG + 萌制作 trailer（含错配 LIVE_N） | 否 | LIVE_<N> 必须匹配实际 mp4 长度 |
| B | 萌制作 JPG + 萌制作 padding + 我们的 mp4 | 否 | JPG 段不是关键 |
| C | 我们的 JPG + 萌制作 padding + 我们的 mp4 | 否 | padding 不是关键 |
| D | 我们的 JPG + padding + 无 B-frame 的 mp4 | 否 | 视频编码细节不是关键 |
| E | 我们的 JPG + padding + ffmpeg copy 的 mz mp4 | 否 | mz 自带的 LIVE footer 被丢弃 |
| F | 我们的 JPG + padding + 转码后的 mz mp4 | 否 | 同上，footer 丢失 |
| **G** | **我们的 JPG + 16 B padding + 我们的 mp4 + 正确 LIVE footer** | **是** | **footer 是唯一缺失件** |

E/F 的失败暴露了核心机制：`ffmpeg -c copy` 不识别这 40 字节非标准 trailer，会在重封装时丢弃 —— 所以必须由调用方在 ffmpeg 输出之后**手动追加**。
