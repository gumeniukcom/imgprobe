# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project follows
[Semantic Versioning](https://semver.org).

## [Unreleased]

### Added

- Initial release. Pure-Go image preflight:
  - `Sniff(head, opts...)` — magic-byte format detection.
  - `Config(data, opts...)` — header-only format + dimensions with EXIF
    orientation and decompression-bomb caps (`ErrTooLarge` / `ErrUnsupported`).
  - Options: `WithMaxPixels`, `WithMaxSide`, `WithOrientation`, `WithFormats`.
  - Built-in formats JPEG/PNG/WEBP (default) and GIF/BMP/TIFF (opt-in); custom
    formats via `NewFormat`.
