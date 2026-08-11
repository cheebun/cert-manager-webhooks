# cert-manager DNS01 Webhook 完整流程

## 整体概览

```
Certificate          （用户声明所需证书）
    ↓
CertificateRequest   （cert-manager 内部创建）
    ↓
Order                （向 ACME CA 发起订单）
    ↓
Challenge × N        （每个域名对应一个）
```

例如 `dnsNames: [example.com, *.example.com]` 会产生 **两个** Challenge，每个域名各一个。

---

## Challenge 生命周期

```
pending → processing → valid
                    ↘ invalid
```

| 状态 | 含义 |
|------|------|
| `pending` | 等待处理 |
| `processing` | Present 已调用，自检进行中 |
| `valid` | ACME 服务器已确认 TXT 记录 |
| `invalid` | 验证失败（超时或出错） |

---

## 详细步骤

### 第一步：Present

cert-manager 调用 webhook 的 `Present` 方法。

```
cert-manager → webhook Present()
    → 调用 DNS 提供商 API：写入 TXT 记录
      _acme-challenge.<domain>  TXT  <key>
```

**关键点：**
- 同一个 Challenge 可能被**多次调用**
- 每次自检失败都会再次触发 Present
- **Present 必须是幂等的**

### 第二步：自检（Self-Check）

cert-manager 轮询 DNS，等待 TXT 记录传播。

```
cert-manager → 递归 DNS 查询
    → _acme-challenge.<domain> TXT ?
        ├── 查到且 value 匹配 → 通过 → 通知 ACME 服务器验证
        └── 查不到 / 不匹配  → 等待 → 再次调用 Present
```

每次重试间隔会指数增长（退避策略）。DNS 传播时间取决于 TTL 和各级 NS 缓存，通常为数秒到数分钟。

### 第三步：ACME 验证

自检通过后，cert-manager 通知 ACME 服务器进行权威验证。

```
cert-manager → ACME 服务器：请验证 _acme-challenge.<domain>
    → ACME 服务器权威查询 DNS
        ├── TXT 存在且匹配 → Challenge valid ✓
        └── TXT 不存在    → Challenge invalid ✗
```

### 第四步：CleanUp

Challenge 变为 `valid` 或 `invalid` 后，cert-manager 调用 CleanUp。

```
cert-manager → webhook CleanUp()
    → 调用 DNS 提供商 API：删除 TXT 记录
```

**关键点：**
- **CleanUp 只在 Challenge 结束后才被调用**
- 如果 Challenge 一直卡在 `pending`/`processing`，CleanUp **永远不会被调用**
- 残留的 TXT 记录会一直留在 DNS 提供商，导致下次签发失败

### 第五步：证书签发

所有 Challenge 都变为 `valid` 后，Order 完成，签好的证书存入 `spec.secretName` 指定的 Secret。

---

## 为什么幂等性至关重要

### Present 会被重复调用

```
时间线：
  t=0s   Present() → dnsAddRecord → 成功
  t=30s  自检失败（DNS 尚未传播）
  t=30s  Present() 再次被调用
         → 不幂等：返回 "already exists" 错误 → Challenge 卡死
         → 幂等：  检测到记录已存在 → return nil → 正常继续
```

### CleanUp 不触发导致记录堆积

```
如果 Challenge 卡住（Present 持续失败）：
  CleanUp 永远不被调用
  → DNS 提供商上 TXT 记录越堆越多
  → 下次签发 Present 再次报 "already exists"
  → 再次卡死 → 无限循环
```

---

## 正确的实现模式

### Present

```
1. 列出该域名下所有 TXT 记录
2. 找到第一条 host == _acme-challenge.<domain> 且 type == TXT 的记录：
   a. value == challenge.Key  → return nil（已正确，幂等）
   b. value != challenge.Key  → dnsUpdateRecord 原地更新 → return
3. 没有找到任何记录 → dnsAddRecord 新建
```

第一条之外的多余残留记录由 CleanUp 负责全部清理。

### CleanUp

```
1. 列出该域名下所有 TXT 记录
2. 删除所有 host == _acme-challenge.<domain> 的记录
   （不判断 value，彻底清理所有残留）
3. 没有找到记录 → return nil（幂等）
```

---

## 各 Provider 实现对比

### Present

| Provider | 预检查 | 记录已存在（value 相同） | 记录已存在（value 不同） | 记录不存在 |
|----------|--------|------------------------|------------------------|-----------|
| **namesilo** | List 所有 TXT | return nil ✅ | 原地 Update 第一条 ✅ | Add ✅ |
| **alidns** | List（精确查询） | return nil ✅ | 原地 Update 第一条 ✅ | Add ✅ |
| **tencent** | List 所有 TXT | return nil ✅ | 原地 Update 第一条 ✅ | Add ✅ |
| **spaceship** | List 所有 TXT | return nil ✅ | PUT `force:true` 覆盖 ✅ | Add ✅ |

### CleanUp

| Provider | 策略 | 删除范围 | 记录不存在时 |
|----------|------|---------|------------|
| **namesilo** | List → 删除所有同名 TXT | `_acme-challenge.<domain>` 下所有 TXT，不区分 value | return nil ✅ |
| **tencent** | List → 删除所有同名 TXT | `_acme-challenge.<domain>` 下所有 TXT，不区分 value | return nil ✅ |
| **alidns** | `DeleteSubDomainRecords`（原子操作） | `_acme-challenge.<domain>` 下所有 TXT，不区分 value | API 不报错 ✅ |
| **spaceship** | List（GET）→ 删除所有同名 TXT | `_acme-challenge.<domain>` 下所有 TXT，不区分 value | return nil ✅ |

---

## 常见故障场景

### "already exists" 死循环

**原因**：Present 不幂等，记录已存在时直接返回错误。
**表现**：Challenge 一直处于 `processing`，CleanUp 不触发，记录持续堆积。
**修复**：写入前先查询；若记录已存在且 value 正确则视为成功；value 不同则原地更新。

### CleanUp "No TXT record found" 循环

**原因**：CleanUp 按 value 精确匹配，但 DNS 提供商 API 有缓存延迟——记录实际已删除，但 API 仍间歇性返回旧数据。
**表现**：CleanUp 返回错误，cert-manager 重试，实际上记录早已不存在。
**修复**：找不到记录时直接返回 nil（幂等 CleanUp）。

### DNS 传播超时

**原因**：TTL 过高或 NS 更新缓慢。
**表现**：自检持续失败，Present 被反复调用。
**修复**：将 `_acme-challenge` 记录的 TTL 设置为较低值（60～300 秒）；确认 webhook 使用了正确的 zone（如 `example.com.` 而非 `com.`）。

### Webhook 配置 zone 错误

**原因**：ClusterIssuer 未配置 `selector.dnsZones`，cert-manager 将 zone 解析为父域（如 `com.`）。
**表现**：Webhook 调用 API 时传入错误的 domain → 返回 `Invalid Domain Syntax`。
**修复**：在 ClusterIssuer 中明确指定 `dnsZones`：

```yaml
solvers:
  - dns01:
      webhook:
        groupName: acme.example.com
        solverName: namesilo
        config:
          apiKey:
            name: namesilo-secret
            key: apiKey
    selector:
      dnsZones:
        - example.com
```
