# Agar Roadmap

## Investigate xiph/flac test_streams for synthetic audio generation

The xiph/flac project has a custom C tool (`src/test_streams/main.c`) that generates
synthetic audio programmatically: sine waves, noise, full-scale deflection patterns,
wasted-bits edge cases, and intentionally malformed files. It outputs to WAV, AIFF,
RF64, W64, and raw PCM across many sample rate / bit depth / channel combinations.

Worth investigating whether we can integrate or adapt this approach in agar to reduce
our dependency on ffmpeg for basic waveform generation, and to gain the malformed /
edge-case file generation that ffmpeg cannot produce.

Reference: https://github.com/xiph/flac/blob/master/src/test_streams/main.c
