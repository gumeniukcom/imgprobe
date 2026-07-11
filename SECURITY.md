# Security Policy

## Supported versions

Only the latest release receives security fixes.

| Version | Supported |
|---|---|
| latest | yes |
| older | no — upgrade |

## Reporting a vulnerability

Please report vulnerabilities **privately** via
[GitHub Security Advisories](https://github.com/gumeniukcom/imgprobe/security/advisories/new)
("Report a vulnerability"). Do not open a public issue for security problems.

You can expect an acknowledgement within **7 days** and a fix or a public
advisory within **90 days** of the report. Credit is given in the advisory
unless you prefer otherwise.

## Threat model and scope

`imgprobe` exists to make one step safe: reading an image's format and
dimensions from untrusted bytes without triggering a decompression bomb.

**In scope:**

- The header-only decode path: probing must never fully decode an image, and a
  declared size over the caps must be rejected before allocation.
- The total-area (`WithMaxPixels`) and per-side (`WithMaxSide`) caps, including
  their safe defaults.
- Magic-byte format detection and the per-call format allowlist.
- Never panicking on arbitrary/truncated input (covered by fuzzing).

**Out of scope — the caller's responsibility:**

- Bounding the raw upload size *before* buffering the bytes handed to
  `Config`/`Sniff`. `imgprobe` operates on bytes already in memory.
- Any subsequent full decode or resize, which must apply its own limits.
- Pixel content, EXIF parsing, or malware in image payloads.

`golang.org/x/image` codec vulnerabilities are upstream; report those to the Go
project, though reports here are welcome and will be routed.
