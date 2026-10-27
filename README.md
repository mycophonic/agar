# Agar

> * a Go testing framework that generates test audio files and ease testing of audio cli tools
> * [a colloidal extract of algae; used especially in culture media](https://en.wikipedia.org/wiki/Agar)

![Agar](logo.jpg)

## Purpose

Testing audio tools requires... test audio files.
Agar provides helpers and scripts to ease generation of such files,
specifically for the purpose of testing edge cases against broken or otherwise
damaged audio streams (clipping, brick-walled, truncated, upsampled, etc).

Agar additionally uses [Tigron](https://github.com/containerd/nerdctl/tree/main/mod/tigron),
a Go test framework specifically designed to test binaries as blackboxes with an expressive syntax, pty handling,
and good debugging information.

Agar is primarily serving Mycophonic audio libraries testing needs, but could
presumably be used by any other audio tool in need of test audio file generation.

Note that Agar relies heavily on ffmpeg for said generation (of course), and you have to install it on your own.

## Usage

TBD. Look at source.

## Development & tests

### Requirements

* [just](https://github.com/casey/just) and limen's toolset: `just` runs every
  task on the pinned tools (Go, the linters), described in `AGENTS.md`
* ffmpeg and ffprobe
* sox_ng (`brew install sox_ng`, which provides the `sox` binary with DSD support)
* metaflac
* other

### Lifecycle

```bash
just lint
just fix
just test
```

`just` alone lists every recipe, the shared `just do …` tree included.
