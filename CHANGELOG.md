# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project follows
[Semantic Versioning](https://semver.org).

## [Unreleased]

### Added

- Benchmarks (`Config`, `Sniff`, bomb-rejection) and
  `TestBombRejectionReadsHeaderOnly`, which fails if rejecting a bomb allocates
  more than header-sized memory — the core security claim, measured.
- Real-file test fixtures under `testdata/`: a WebP (Go has no webp encoder, so
  this is the only way to cover the webp decode path) and a big-endian TIFF
  (covers the `MM` branch that `x/image/tiff.Encode` can't produce).
- CodeQL (SAST) and Dependabot workflows.

## [0.1.0] - 2026-07-11

### Added

- Initial release. Pure-Go image preflight:
  - `Sniff(head, opts...)` — magic-byte format detection.
  - `Config(data, opts...)` — header-only format + dimensions with EXIF
    orientation and decompression-bomb caps (`ErrTooLarge` / `ErrUnsupported`).
  - Options: `WithMaxPixels`, `WithMaxSide`, `WithOrientation`, `WithFormats`.
  - Built-in formats JPEG/PNG/WEBP (default) and GIF/BMP/TIFF (opt-in); custom
    formats via `NewFormat`.
