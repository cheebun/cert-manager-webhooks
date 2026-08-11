# cert-manager DNS01 Webhook Flow

## Overview

```
Certificate          (user declares desired certificate)
    ↓
CertificateRequest   (cert-manager creates internally)
    ↓
Order                (ACME order placed with CA)
    ↓
Challenge × N        (one per domain name)
```

A `Certificate` resource with `dnsNames: [example.com, *.example.com]` produces
**two** Challenges — one for each domain.

---

## Challenge Lifecycle

```
pending → processing → valid
                    ↘ invalid
```

| State | Meaning |
|-------|---------|
| `pending` | Waiting to be processed |
| `processing` | Present called, self-check in progress |
| `valid` | ACME server confirmed the TXT record |
| `invalid` | Verification failed (expired or error) |

---

## Step-by-Step Flow

### 1. Present

cert-manager calls the webhook's `Present` method.

```
cert-manager → webhook Present()
    → DNS provider API: write TXT record
      _acme-challenge.<domain>  TXT  <key>
```

- May be called **multiple times** for the same Challenge
- Every failed self-check triggers another `Present` call
- **Present must be idempotent**

### 2. Self-Check

cert-manager polls DNS until the TXT record propagates.

```
cert-manager → recursive DNS query
    → _acme-challenge.<domain> TXT ?
        ├── found & value matches → pass → notify ACME server
        └── not found / mismatch  → wait → call Present again
```

Polling interval increases with each retry (backoff).
DNS propagation can take seconds to minutes depending on the provider's TTL and
nameserver caching.

### 3. ACME Verification

Once self-check passes, cert-manager notifies the ACME server to verify.

```
cert-manager → ACME server: please verify _acme-challenge.<domain>
    → ACME server queries DNS authoritatively
        ├── TXT found & matches → Challenge valid ✓
        └── not found           → Challenge invalid ✗
```

### 4. CleanUp

After the Challenge reaches `valid` or `invalid`, cert-manager calls `CleanUp`.

```
cert-manager → webhook CleanUp()
    → DNS provider API: delete TXT record(s)
```

- **CleanUp is only called after the Challenge is resolved**
- If a Challenge stays `pending`/`processing` indefinitely, CleanUp is never
  called — stale TXT records accumulate on the DNS provider

### 5. Certificate Issuance

Once all Challenges are `valid`, the Order completes and the signed certificate
is stored in the Secret named by `spec.secretName`.

---

## Why Idempotency Matters

### Present — called repeatedly

```
Timeline:
  t=0   Present() → dnsAddRecord → success
  t=30s self-check fails (DNS not propagated yet)
  t=30s Present() called again
        → if not idempotent: "already exists" error → Challenge stuck
        → if idempotent:     detect existing record → return nil → OK
```

### CleanUp — stale record accumulation

```
If Challenge gets stuck (Present keeps failing):
  CleanUp is never called
  → TXT records pile up on DNS provider
  → Next issuance attempt hits "already exists" on Present
  → Stuck again → infinite loop
```

---

## Correct Implementation Pattern

### Present

```
1. List all TXT records for the domain
2. Find the first record where host == _acme-challenge.<domain> and type == TXT:
   a. value == challenge.Key  → return nil (already correct, idempotent)
   b. value != challenge.Key  → dnsUpdateRecord in place → return
3. No record found → dnsAddRecord
```

Any extra stale records beyond the first are left for CleanUp to remove.

### CleanUp

```
1. List all TXT records for the domain
2. Delete ALL records where host == _acme-challenge.<domain>
   (regardless of value — cleans up any residue)
3. If none found → return nil (idempotent)
```

---

## Provider Comparison

### Present

| Provider | Pre-check | Record exists (same value) | Record exists (different value) | No record |
|----------|-----------|---------------------------|--------------------------------|-----------|
| **namesilo** | List all TXT | return nil ✅ | Update first record in place ✅ | Add ✅ |
| **alidns** | List (exact query) | return nil ✅ | Update first record in place ✅ | Add ✅ |
| **tencent** | List all TXT | return nil ✅ | Update first record in place ✅ | Add ✅ |
| **spaceship** | List all TXT | return nil ✅ | PUT `force:true` overwrites ✅ | Add ✅ |

### CleanUp

| Provider | Strategy | Scope | Record not found |
|----------|----------|-------|-----------------|
| **namesilo** | List → delete all same-host TXT | All records under `_acme-challenge.<domain>`, regardless of value | return nil ✅ |
| **tencent** | List → delete all same-host TXT | All records under `_acme-challenge.<domain>`, regardless of value | return nil ✅ |
| **alidns** | `DeleteSubDomainRecords` (atomic) | All records under `_acme-challenge.<domain>`, regardless of value | API does not error ✅ |
| **spaceship** | List (GET) → DELETE all same-host TXT | All records under `_acme-challenge.<domain>`, regardless of value | return nil ✅ |

---

## Common Failure Scenarios

### "already exists" loop

**Cause**: Present is not idempotent — returns error when record already exists.
**Effect**: Challenge stays `processing`, CleanUp never called, records pile up.
**Fix**: Check for existing record before adding; treat "already exists" as success
or update in place.

### "No TXT record found" in CleanUp loop

**Cause**: CleanUp looks for exact value match, but DNS provider API has cache lag —
record appears deleted via API but old data is still returned intermittently.
**Effect**: CleanUp returns error, cert-manager retries, record may already be gone.
**Fix**: Treat "record not found" as success (idempotent CleanUp).

### DNS propagation timeout

**Cause**: DNS TTL is too high or nameservers are slow to update.
**Effect**: Self-check keeps failing, Present is called repeatedly.
**Fix**: Set low TTL (60–300s) on `_acme-challenge` records; ensure webhook
uses correct zone (not `com.` but `example.com.`).

### Wrong zone in webhook config

**Cause**: ClusterIssuer `solverConfig` does not specify `selector.dnsZones`,
cert-manager resolves to a parent zone (e.g. `com.`).
**Effect**: Webhook calls API with wrong domain → `Invalid Domain Syntax` error.
**Fix**: Add explicit `dnsZones` selector in the ClusterIssuer.

```yaml
solvers:
  - dns01:
      webhook:
        groupName: acme.namesilo.com
        solverName: namesilo
        config:
          apiKey:
            name: namesilo-secret
            key: apiKey
    selector:
      dnsZones:
        - example.com
```
