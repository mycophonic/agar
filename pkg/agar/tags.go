package agar

import (
	"path/filepath"

	"github.com/containerd/nerdctl/mod/tigron/test"
)

// AddTag adds a tag to a FLAC file using metaflac.
// This allows adding multiple values for the same tag key.
func AddTag(helpers test.Helpers, path, key, value string) {
	helpers.T().Helper()

	helpers.Custom(metaflacBinary, "--set-tag="+key+"="+value, path).Run(&test.Expected{})
}

// RemoveTag removes all instances of a tag from a FLAC file using metaflac.
func RemoveTag(helpers test.Helpers, path, key string) {
	helpers.T().Helper()

	helpers.Custom(metaflacBinary, "--remove-tag="+key, path).Run(&test.Expected{})
}

// SetTag sets a tag value, removing any existing values for that key first.
func SetTag(helpers test.Helpers, path, key, value string) {
	helpers.T().Helper()

	RemoveTag(helpers, path, key)
	AddTag(helpers, path, key, value)
}

// TaggedFLAC returns path to FLAC with standard metadata tags.
func TaggedFLAC(data test.Data, helpers test.Helpers) string {
	helpers.T().Helper()

	return generate(helpers, filepath.Join(data.Temp().Dir(), "tagged.flac"), []string{
		"-f", "lavfi", "-i", "sine=frequency=440:duration=" + shortDuration,
		"-af", "pan=stereo|c0=c0|c1=c0,volume=-6dB",
		"-ar", "44100", "-sample_fmt", "s16",
		"-metadata", "artist=Test Artist",
		"-metadata", "album=Test Album",
		"-metadata", "title=Test Title",
		"-metadata", "date=2024",
		"-metadata", "tracknumber=1",
		"-metadata", "genre=Electronic",
	})
}

// TaggedFLACMultiArtist returns path to FLAC with multiple artist values.
// Uses metaflac to add duplicate ARTIST tags since FFmpeg doesn't support this.
func TaggedFLACMultiArtist(data test.Data, helpers test.Helpers) string {
	helpers.T().Helper()

	path := generate(helpers, filepath.Join(data.Temp().Dir(), "tagged-multi-artist.flac"), []string{
		"-f", "lavfi", "-i", "sine=frequency=440:duration=" + shortDuration,
		"-af", "pan=stereo|c0=c0|c1=c0,volume=-6dB",
		"-ar", "44100", "-sample_fmt", "s16",
		"-metadata", "artist=Artist One",
		"-metadata", "album=Collaboration Album",
	})

	// Add second artist using metaflac
	AddTag(helpers, path, "ARTIST", "Artist Two")

	return path
}
