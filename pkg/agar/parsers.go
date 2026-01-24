package agar

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Sentinel errors for unsupported formats.
var (
	ErrMP3NotSupported  = errors.New("MP3/ID3v2 parsing not yet supported")
	ErrOGGNotSupported  = errors.New("OGG Vorbis parsing not yet supported")
	ErrOpusNotSupported = errors.New("opus parsing not yet supported")
)

// ParsedTags holds metadata parsed from a native tool.
type ParsedTags struct {
	// Text tags as key -> []values (supports multiple values per key).
	Text map[string][]string

	// Track number and total.
	Track      int
	TrackTotal int

	// Disc number and total.
	Disc      int
	DiscTotal int

	// Picture count.
	PictureCount int
}

// NewParsedTags creates a new empty ParsedTags.
func NewParsedTags() *ParsedTags {
	return &ParsedTags{
		Text: make(map[string][]string),
	}
}

// atomicParsleyAtomRE matches AtomicParsley -t output lines for standard atoms.
// Example: Atom "©nam" contains: Test Title.
var atomicParsleyAtomRE = regexp.MustCompile(`^Atom "([^"]+)" contains: (.*)$`)

// atomicParsleyFreeformRE matches AtomicParsley -t output lines for freeform (----) atoms.
// Example: Atom "----" [com.apple.iTunes;MusicBrainz Album Id] contains: 69af009e-1e38-447e-a43b-203fa5f5095f.
var atomicParsleyFreeformRE = regexp.MustCompile(`^Atom "----" \[([^;]+);([^\]]+)\] contains: (.*)$`)

// atomicParsleyArtworkRE matches AtomicParsley artwork atoms.
// Example: Atom "covr" contains: 1 piece of artwork.
var atomicParsleyArtworkRE = regexp.MustCompile(`^Atom "covr" contains: (\d+) piece`)

// ParseAtomicParsley runs AtomicParsley -t on the file and parses output.
func ParseAtomicParsley(ctx context.Context, filePath string) (*ParsedTags, error) {
	cmd := exec.CommandContext(ctx, atomicParsleyBinary, filePath, "-t")

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("atomicparsley failed: %w\nstderr: %s", err, stderr.String())
	}

	output := stdout.Bytes()

	tags := NewParsedTags()
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		// Skip BOM if present
		line = strings.TrimPrefix(line, "\ufeff")

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for artwork
		if matches := atomicParsleyArtworkRE.FindStringSubmatch(line); matches != nil {
			tags.PictureCount, _ = strconv.Atoi(matches[1])

			continue
		}

		// Check for freeform (----) atoms first since they also start with Atom "
		if matches := atomicParsleyFreeformRE.FindStringSubmatch(line); matches != nil {
			// domain := matches[1] // e.g., "com.apple.iTunes"
			name := matches[2] // e.g., "MusicBrainz Album Id"
			value := matches[3]

			// Convert to semantic name to match gill's representation
			key := freeformToSemanticName(name)
			tags.Text[key] = append(tags.Text[key], value)

			continue
		}

		// Check for regular atoms
		if matches := atomicParsleyAtomRE.FindStringSubmatch(line); matches != nil {
			atomName := matches[1]
			value := matches[2]

			// Handle special atoms
			switch atomName {
			case "trkn":
				tags.Track, tags.TrackTotal = parsePairValue(value)
				// Also add to Text in gill's format (N/M instead of N of M)
				tags.Text["tracknumber"] = append(
					tags.Text["tracknumber"],
					formatPairValue(tags.Track, tags.TrackTotal),
				)
			case "disk":
				tags.Disc, tags.DiscTotal = parsePairValue(value)
				// Also add to Text in gill's format (N/M instead of N of M)
				tags.Text["discnumber"] = append(tags.Text["discnumber"], formatPairValue(tags.Disc, tags.DiscTotal))
			default:
				// Normalize atom name to match go-mp4's representation
				// AtomicParsley uses © directly, go-mp4 uses (c)
				normalizedName := normalizeAtomName(atomName)
				// Convert to semantic name to match gill's representation
				semanticName := atomToSemanticName(normalizedName)
				tags.Text[semanticName] = append(tags.Text[semanticName], value)
			}

			continue
		}

		// Unmatched lines are silently ignored (e.g., metadata headers)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning atomicparsley output: %w", err)
	}

	return tags, nil
}

// ParseMetaflac runs metaflac --export-tags-to=- on the file and parses output.
func ParseMetaflac(ctx context.Context, filePath string) (*ParsedTags, error) {
	cmd := exec.CommandContext(ctx, metaflacBinary, "--export-tags-to=-", filePath)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("metaflac failed: %w\nstderr: %s", err, stderr.String())
	}

	output := stdout.Bytes()

	tags := NewParsedTags()
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "="); idx > 0 {
			key := line[:idx]
			value := line[idx+1:]

			// Vorbis comment keys are case-insensitive, normalize to uppercase
			upperKey := strings.ToUpper(key)

			// Handle special tags
			switch upperKey {
			case "TRACKNUMBER":
				tags.Track, _ = parsePairValue(value)
				// Add to Text in gill's format
				tags.Text["tracknumber"] = append(tags.Text["tracknumber"], value)
			case "TRACKTOTAL", "TOTALTRACKS":
				tags.TrackTotal, _ = strconv.Atoi(value)
				// Also store as semantic name
				semanticKey := vorbisToSemanticName(upperKey)
				tags.Text[semanticKey] = append(tags.Text[semanticKey], value)
			case "DISCNUMBER":
				tags.Disc, _ = parsePairValue(value)
				// Add to Text in gill's format
				tags.Text["discnumber"] = append(tags.Text["discnumber"], value)
			case "DISCTOTAL", "TOTALDISCS":
				tags.DiscTotal, _ = strconv.Atoi(value)
				// Also store as semantic name
				semanticKey := vorbisToSemanticName(upperKey)
				tags.Text[semanticKey] = append(tags.Text[semanticKey], value)
			default:
				// Convert to semantic name
				semanticKey := vorbisToSemanticName(upperKey)
				tags.Text[semanticKey] = append(tags.Text[semanticKey], value)
			}
		}
	}

	// Get picture count separately
	tags.PictureCount = countMetaflacPictures(ctx, filePath)

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning metaflac output: %w", err)
	}

	return tags, nil
}

// countMetaflacPictures counts PICTURE blocks in a FLAC file.
func countMetaflacPictures(ctx context.Context, filePath string) int {
	cmd := exec.CommandContext(ctx, metaflacBinary, "--list", "--block-type=PICTURE", filePath)

	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Count "type: 6 (PICTURE)" lines
	count := 0
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "type: 6 (PICTURE)") {
			count++
		}
	}

	return count
}

// ParseID3v2 is a stub that logs a warning - MP3 not yet supported.
func ParseID3v2(filePath string) (*ParsedTags, error) {
	return nil, fmt.Errorf("%w: %s", ErrMP3NotSupported, filePath)
}

// ParseVorbisComment is a stub that logs a warning - OGG Vorbis not yet supported.
func ParseVorbisComment(filePath string) (*ParsedTags, error) {
	return nil, fmt.Errorf("%w: %s", ErrOGGNotSupported, filePath)
}

// ParseOpusTags is a stub that logs a warning - Opus not yet supported.
func ParseOpusTags(filePath string) (*ParsedTags, error) {
	return nil, fmt.Errorf("%w: %s", ErrOpusNotSupported, filePath)
}

// formatPairValue formats a number/total pair as "N/M" string.
func formatPairValue(num, total int) string {
	if total > 0 {
		return fmt.Sprintf("%d/%d", num, total)
	}

	return strconv.Itoa(num)
}

// parsePairValue parses "N/M" or "N of M" or just "N" into number and total.
func parsePairValue(value string) (num, total int) {
	// Try "N/M" format
	if parts := strings.Split(value, "/"); len(parts) == 2 {
		num, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
		total, _ = strconv.Atoi(strings.TrimSpace(parts[1]))

		return num, total
	}

	// Try "N of M" format
	if parts := strings.Split(value, " of "); len(parts) == 2 {
		num, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
		total, _ = strconv.Atoi(strings.TrimSpace(parts[1]))

		return num, total
	}

	// Just a number
	num, _ = strconv.Atoi(strings.TrimSpace(value))

	return num, 0
}

// normalizeAtomName converts AtomicParsley's atom names to match go-mp4's representation.
// AtomicParsley uses © (0xa9) directly, while go-mp4 represents it as "(c)".
func normalizeAtomName(name string) string {
	// Replace © (U+00A9) with "(c)"
	return strings.ReplaceAll(name, "\u00a9", "(c)")
}

// mp4AtomToSemantic maps MP4 atom four-char codes to semantic names.
// This matches how gill normalizes these atoms.
//
//nolint:gochecknoglobals // lookup table
var mp4AtomToSemantic = map[string]string{
	"(c)nam": "title",
	"(c)ART": "artist",
	"(c)alb": "album",
	"(c)day": "date",
	"(c)cmt": "comment",
	"(c)wrt": "composer",
	"(c)gen": "genre",
	"(c)grp": "grouping",
	"(c)lyr": "lyrics",
	"(c)wrk": "work",
	"(c)mvn": "movement",
	"aART":   "albumartist",
	"soar":   "artistsort",
	"soaa":   "albumartistsort",
	"soco":   "composersort",
	"sonm":   "titlesort",
	"soal":   "albumsort",
	"cprt":   "copyright",
	"desc":   "description",
	"ldes":   "longdescription",
	"tvsh":   "tvshow",
	"tven":   "tvepisodeid",
	"tvnn":   "tvnetwork",
	"tvsn":   "tvseason",
	"tves":   "tvepisode",
	"purd":   "purchasedate",
	"pcst":   "podcast",
	"catg":   "category",
	"keyw":   "keyword",
	"purl":   "podcasturl",
	"egid":   "episodeguid",
	"stik":   "mediatype",
	"hdvd":   "hd",
	"rtng":   "rating",
	"pgap":   "gapless",
	"cpil":   "compilation",
	"tmpo":   "tempo",
	"----":   "", // Freeform, handled separately
}

// freeformNameToSemantic maps freeform tag names to semantic names.
// The keys are the name portion (after the semicolon) in uppercase for case-insensitive matching.
//
//nolint:gochecknoglobals // lookup table
var freeformNameToSemantic = map[string]string{
	"MUSICBRAINZ ALBUM RELEASE COUNTRY": "releasecountry",
	"MUSICBRAINZ ALBUM ID":              "musicbrainz_albumid",
	"MUSICBRAINZ ARTIST ID":             "musicbrainz_artistid",
	"MUSICBRAINZ ALBUM ARTIST ID":       "musicbrainz_albumartistid",
	"MUSICBRAINZ TRACK ID":              "musicbrainz_recordingid",
	"MUSICBRAINZ RELEASE TRACK ID":      "musicbrainz_releasetrackid",
	"MUSICBRAINZ RELEASE GROUP ID":      "musicbrainz_releasegroupid",
	"MUSICBRAINZ WORK ID":               "musicbrainz_workid",
	"ACOUSTID ID":                       "acoustid_id",
	"MUSICBRAINZ ALBUM TYPE":            "releasetype",
	"MUSICBRAINZ ALBUM STATUS":          "releasestatus",
	"ASIN":                              "asin",
	"LABEL":                             "label",
	"CATALOGNUMBER":                     "catalognumber",
	"MEDIA":                             "media",
	"SCRIPT":                            "script",
	"LANGUAGE":                          "language",
	"ORIGINALDATE":                      "originaldate",
	"ORIGINALYEAR":                      "originalyear",
	"ARTISTS":                           "artists",
	"ARRANGER":                          "arranger",
	"BARCODE":                           "barcode",
	"ISRC":                              "isrc",
}

// vorbisToSemantic maps Vorbis comment tag names (UPPERCASE) to semantic names.
// Based on MusicBrainz Picard tag mapping:
// https://picard-docs.musicbrainz.org/downloads/MusicBrainz_Picard_Tag_Map.html
// IMPORTANT: MUSICBRAINZ_TRACKID is the Recording ID, not the track ID!
//
//nolint:gochecknoglobals // lookup table
var vorbisToSemantic = map[string]string{
	// Standard tags
	"ALBUM":           "album",
	"ALBUMARTIST":     "albumartist",
	"ALBUMARTISTSORT": "albumartistsort",
	"ARTIST":          "artist",
	"ARTISTSORT":      "artistsort",
	"ARTISTS":         "artists",
	"ASIN":            "asin",
	"BARCODE":         "barcode",
	"CATALOGNUMBER":   "catalognumber",
	"COMMENT":         "comment",
	"COMPILATION":     "compilation",
	"COMPOSER":        "composer",
	"COMPOSERSORT":    "composersort",
	"COPYRIGHT":       "copyright",
	"DATE":            "date",
	"DISCSUBTITLE":    "discsubtitle",
	"DISCTOTAL":       "disctotal",
	"TOTALDISCS":      "totaldiscs",
	"GENRE":           "genre",
	"ISRC":            "isrc",
	"LABEL":           "label",
	"LANGUAGE":        "language",
	"LYRICS":          "lyrics",
	"MEDIA":           "media",
	"ORIGINALDATE":    "originaldate",
	"ORIGINALYEAR":    "originalyear",
	"PERFORMER":       "performer",
	"RELEASECOUNTRY":  "releasecountry",
	"RELEASESTATUS":   "releasestatus",
	"RELEASETYPE":     "releasetype",
	"SCRIPT":          "script",
	"TITLE":           "title",
	"TITLESORT":       "titlesort",
	"TOTALTRACKS":     "totaltracks",
	"TRACKTOTAL":      "tracktotal",
	"WORK":            "work",
	"WRITER":          "writer",
	"ARRANGER":        "arranger",
	// MusicBrainz IDs - CRITICAL: Vorbis MUSICBRAINZ_TRACKID = Recording ID!
	"MUSICBRAINZ_TRACKID":        "musicbrainz_recordingid",    // Recording MBID (confusing name!)
	"MUSICBRAINZ_RELEASETRACKID": "musicbrainz_releasetrackid", // Release Track MBID
	"MUSICBRAINZ_ALBUMID":        "musicbrainz_albumid",
	"MUSICBRAINZ_ARTISTID":       "musicbrainz_artistid",
	"MUSICBRAINZ_ALBUMARTISTID":  "musicbrainz_albumartistid",
	"MUSICBRAINZ_RELEASEGROUPID": "musicbrainz_releasegroupid",
	"MUSICBRAINZ_WORKID":         "musicbrainz_workid",
	"ACOUSTID_ID":                "acoustid_id",
}

// vorbisToSemanticName converts a Vorbis comment tag name to a semantic name.
func vorbisToSemanticName(vorbisKey string) string {
	if semantic, ok := vorbisToSemantic[vorbisKey]; ok {
		return semantic
	}

	// Default: lowercase the key
	return strings.ToLower(vorbisKey)
}

// atomToSemanticName converts a raw MP4 atom name to a semantic name.
func atomToSemanticName(atomName string) string {
	if semantic, ok := mp4AtomToSemantic[atomName]; ok && semantic != "" {
		return semantic
	}

	return atomName
}

// freeformToSemanticName converts a freeform tag name to a semantic name.
func freeformToSemanticName(name string) string {
	upperName := strings.ToUpper(name)
	if semantic, ok := freeformNameToSemantic[upperName]; ok {
		return semantic
	}

	// Default: lowercase and replace spaces with underscores
	return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
}
