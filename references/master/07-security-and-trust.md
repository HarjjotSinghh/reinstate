# Security & Trust (Reinstate)

## Why this is existential

**Reinstate** will touch some of the most sensitive material on a developer machine:

- Proprietary source fragments in transcripts  
- Customer data in tool outputs  
- Shell commands and logs  
- MCP configs (including startup commands → **arbitrary code**)  
- Skills and instruction files  
- Git metadata  
- Secrets accidentally printed into sessions  
- API keys in env dumps / error messages  

A bad implementation is an efficient **source-code exfiltration platform**. Enterprises and power users will not adopt without a credible threat model.

---

## Minimum trust model (consensus across sources)

| Requirement | Why | Precedents / sources |
|-------------|-----|----------------------|
| **Client-side / E2EE for session content** | Server must not read raw code/transcripts | Kimi, ChatGPT, Claude, Gemini deep, Perplexity |
| **Secret redaction before encryption** | E2EE doesn’t help compromised devices, sharing, local indexes | SpecStory, Kontinuo, ChatGPT |
| **Secret references, not values** | Sync `1password://…` not `ghp_…` | Warp MCP secret separation |
| **Never sync auth tokens / auth.json** | Account-bound / transferable abuse | codexSync refuses; Claude research |
| **Session IDs ≠ authentication** | Hijacking / event injection | MCP security best practices |
| **Configs/skills are executable code** | MCP startup commands, skill scripts | MCP docs; ChatGPT security section |
| **Device enrollment & revocation** | Cross-device without device trust is dangerous | Remote Control trusted devices; ChatGPT |
| **Consent before applying imported hooks/startup** | One-click config = ACE risk | MCP security |
| **Local-first / BYO storage option** | Trust + vendor absorption hedge | claude-sync, Kimi |
| **Opt-in per repo / scope** | Not all projects should leave the machine | Claude FRs; product design |
| **Atomic writes + backups** | Corrupting session history destroys reputation | claude-sync backup pattern |

---

## What the server should ideally see

With strong E2EE:

- User ID / org ID  
- Device ID  
- Encrypted blob IDs and sizes  
- Sync cursors / versions  
- Minimal routing metadata  

Not:

- Plaintext transcripts  
- Plaintext code  
- Raw secrets  
- Decryptable content without user keys  

---

## Secret handling pipeline

```text
Capture → redact common secret patterns → encrypt → upload
Restore → decrypt → path rewrite → quarantine executable config → user consent → apply
```

Redaction targets (non-exhaustive):

- Cloud API keys, GitHub/GitLab tokens  
- JWT / bearer headers  
- Connection strings with passwords  
- Private keys  
- `.env` style KEY=value when high entropy  

Encryption does **not** replace redaction for:

- Compromised peer devices  
- Accidental share links  
- Local search indexes  
- Exported bundles  

---

## MCP / capability plane risks

MCP guidance (cited by ChatGPT deep research):

- Do not use sessions for authentication  
- Secure non-deterministic session IDs; bind to user identity  
- Validate inbound requests; defend DNS rebinding / SSRF  
- Streamable HTTP: validate `Origin`; bind local servers to localhost  
- No token passthrough; servers accept only tokens issued for them  
- One-click install flows must show consent for startup commands  

**Reinstate rules:**

- Diff imported MCP/hooks/skills before apply  
- Track provenance (which device/user introduced a server)  
- Quarantine newly imported executable content  
- Require reauthorization for external services  
- Never silently execute something only because another device had it  

---

## Device management (build early)

- Device enrollment  
- Device revocation  
- Key rotation  
- Remote session invalidation  
- Optional OS keychain / biometric unlock for local keys  
- Audit history (teams)  

Bolting crypto onto an unencrypted protocol later is “engineering root canal” (ChatGPT).

---

## Transport / local server hardening

If Reinstate runs a local daemon or HTTP surface:

- Localhost binding by default  
- Origin validation  
- Auth on all non-local connections  
- No `0.0.0.0` exposure without explicit opt-in  

---

## Grok Build / telemetry special case

Research claim: Grok Build may upload session traces to GCS (`grok-code-session-traces`) even when “improve model” is off.  

**Reinstate policy options:**

- Delay Grok adapter until privacy is clean  
- Or strip/quarantine vendor telemetry content before storing in Reinstate  
- Never re-upload user sessions to third-party training buckets  

---

## Trust & open source (strategic link)

ChatGPT post-naming recommendation for Reinstate:

> Open-source the **local engine and adapters**. Keep **hosted sync / cloud control plane** proprietary if commercializing.

Developers who grant filesystem + agent access will demand to inspect:

- Redaction  
- Encryption  
- What leaves the machine  
- That restore doesn’t execute untrusted config  

See [09-product-positioning.md](./09-product-positioning.md).

---

## Security marketing (not optional)

Primary commercial pillar (Gemini deep research): **zero-knowledge / E2EE**.  
Without it, enterprise and serious indie adoption fails regardless of features.

Publish:

- Threat model doc  
- What is encrypted vs metadata  
- What is never synced  
- How keys work across devices  
- How to self-host / BYO bucket  

Match the transparency posture of tools like claude-sync (documented crypto).
