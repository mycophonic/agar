//go:build darwin && cgo

/*
   Copyright Mycophonic.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package coreaudio

/*
   #cgo LDFLAGS: -framework AudioToolbox -framework CoreFoundation

   #include <AudioToolbox/AudioToolbox.h>
   #include <CoreFoundation/CoreFoundation.h>
   #include <stdlib.h>
   #include <string.h>
   #include <unistd.h>

   // mem_reader provides AudioFileOpenWithCallbacks access to in-memory data.
   typedef struct {
    const char *data;
    int64_t     size;
   } mem_reader;

   static OSStatus mem_read_proc(void *inClientData, SInt64 inPosition, UInt32 requestCount,
                                 void *buffer, UInt32 *actualCount) {
    mem_reader *r = (mem_reader *)inClientData;
    if (inPosition >= r->size) {
        *actualCount = 0;
        return noErr;
    }
    int64_t avail = r->size - inPosition;
    UInt32 toRead = (UInt32)(avail < requestCount ? avail : requestCount);
    memcpy(buffer, r->data + inPosition, toRead);
    *actualCount = toRead;
    return noErr;
   }

   static SInt64 mem_get_size_proc(void *inClientData) {
    return ((mem_reader *)inClientData)->size;
   }

   // decode_result holds the output of coreaudio_decode.
   typedef struct {
    char    *pcm;
    int64_t  pcm_size;
    int      sample_rate;
    int      bit_depth;
    int      channels;
    char     error[256];
   } decode_result;

   // coreaudio_decode decodes audio data from memory using AudioToolbox.
   static int coreaudio_decode(const char *input, int64_t input_size, decode_result *result) {
    memset(result, 0, sizeof(*result));

    mem_reader reader = { .data = input, .size = input_size };

    AudioFileID audioFile = NULL;
    OSStatus status = AudioFileOpenWithCallbacks(
        &reader, mem_read_proc, NULL, mem_get_size_proc, NULL,
        0, &audioFile);
    if (status != noErr) {
        snprintf(result->error, sizeof(result->error),
            "AudioFileOpenWithCallbacks failed (OSStatus %d)", (int)status);
        return 1;
    }

    ExtAudioFileRef extFile = NULL;
    status = ExtAudioFileWrapAudioFileID(audioFile, false, &extFile);
    if (status != noErr) {
        snprintf(result->error, sizeof(result->error),
            "ExtAudioFileWrapAudioFileID failed (OSStatus %d)", (int)status);
        AudioFileClose(audioFile);
        return 1;
    }

    // Query source format.
    AudioStreamBasicDescription srcFormat;
    UInt32 propSize = sizeof(srcFormat);
    status = ExtAudioFileGetProperty(extFile, kExtAudioFileProperty_FileDataFormat, &propSize, &srcFormat);
    if (status != noErr) {
        snprintf(result->error, sizeof(result->error),
            "cannot read source format (OSStatus %d)", (int)status);
        ExtAudioFileDispose(extFile);
        AudioFileClose(audioFile);
        return 1;
    }

    // Determine output bit depth.
    UInt32 outBits = srcFormat.mBitsPerChannel;
    if (outBits == 0 && srcFormat.mFormatID == kAudioFormatAppleLossless) {
        switch (srcFormat.mFormatFlags) {
        case kAppleLosslessFormatFlag_16BitSourceData: outBits = 16; break;
        case kAppleLosslessFormatFlag_20BitSourceData: outBits = 20; break;
        case kAppleLosslessFormatFlag_24BitSourceData: outBits = 24; break;
        case kAppleLosslessFormatFlag_32BitSourceData: outBits = 32; break;
        default: outBits = 16; break;
        }
    }
    if (outBits == 0) outBits = 16;

    // Store format info in result.
    result->sample_rate = (int)srcFormat.mSampleRate;
    result->bit_depth = (int)outBits;
    result->channels = (int)srcFormat.mChannelsPerFrame;

    // 20-bit source is output as 24-bit.
    UInt32 clientBits = outBits;
    if (clientBits == 20) clientBits = 24;
    UInt32 bytesPerSample = clientBits / 8;

    // Set client format: interleaved signed LE PCM.
    AudioStreamBasicDescription clientFormat;
    memset(&clientFormat, 0, sizeof(clientFormat));
    clientFormat.mSampleRate       = srcFormat.mSampleRate;
    clientFormat.mFormatID         = kAudioFormatLinearPCM;
    clientFormat.mFormatFlags      = kAudioFormatFlagIsSignedInteger | kAudioFormatFlagIsPacked;
    clientFormat.mBitsPerChannel   = clientBits;
    clientFormat.mChannelsPerFrame = srcFormat.mChannelsPerFrame;
    clientFormat.mBytesPerFrame    = bytesPerSample * srcFormat.mChannelsPerFrame;
    clientFormat.mFramesPerPacket  = 1;
    clientFormat.mBytesPerPacket   = clientFormat.mBytesPerFrame;

    status = ExtAudioFileSetProperty(extFile, kExtAudioFileProperty_ClientDataFormat,
        sizeof(clientFormat), &clientFormat);
    if (status != noErr) {
        snprintf(result->error, sizeof(result->error),
            "cannot set client format (OSStatus %d)", (int)status);
        ExtAudioFileDispose(extFile);
        AudioFileClose(audioFile);
        return 1;
    }

    // Total frame count.
    SInt64 totalFrames = 0;
    propSize = sizeof(totalFrames);
    status = ExtAudioFileGetProperty(extFile, kExtAudioFileProperty_FileLengthFrames, &propSize, &totalFrames);
    if (status != noErr || totalFrames <= 0) {
        snprintf(result->error, sizeof(result->error),
            "cannot determine frame count (OSStatus %d, frames %lld)", (int)status, totalFrames);
        ExtAudioFileDispose(extFile);
        AudioFileClose(audioFile);
        return 1;
    }

    // Allocate output buffer.
    int64_t pcmSize = totalFrames * clientFormat.mBytesPerFrame;
    char *pcm = (char *)malloc(pcmSize);
    if (!pcm) {
        snprintf(result->error, sizeof(result->error), "out of memory (%lld bytes)", pcmSize);
        ExtAudioFileDispose(extFile);
        AudioFileClose(audioFile);
        return 1;
    }

    // Decode loop.
    const UInt32 framesPerRead = 4096;
    int64_t offset = 0;
    SInt64 framesDecoded = 0;

    while (framesDecoded < totalFrames) {
        SInt64 remaining = totalFrames - framesDecoded;
        UInt32 frameCount = (remaining < framesPerRead) ? (UInt32)remaining : framesPerRead;

        AudioBufferList bufList;
        bufList.mNumberBuffers = 1;
        bufList.mBuffers[0].mNumberChannels = srcFormat.mChannelsPerFrame;
        bufList.mBuffers[0].mDataByteSize   = frameCount * clientFormat.mBytesPerFrame;
        bufList.mBuffers[0].mData           = pcm + offset;

        status = ExtAudioFileRead(extFile, &frameCount, &bufList);
        if (status != noErr) {
            snprintf(result->error, sizeof(result->error),
                "ExtAudioFileRead failed (OSStatus %d)", (int)status);
            free(pcm);
            ExtAudioFileDispose(extFile);
            AudioFileClose(audioFile);
            return 1;
        }
        if (frameCount == 0) break;

        offset += frameCount * clientFormat.mBytesPerFrame;
        framesDecoded += frameCount;
    }

    ExtAudioFileDispose(extFile);
    AudioFileClose(audioFile);

    result->pcm = pcm;
    result->pcm_size = offset;
    return 0;
   }

   // encode_result holds the output of coreaudio_encode.
   typedef struct {
    char    *m4a;
    int64_t  m4a_size;
    char     error[256];
   } encode_result;

   // coreaudio_encode encodes PCM to ALAC M4A using AudioToolbox.
   // Returns 0 on success, non-zero on error.
   static int coreaudio_encode(const char *pcm, int64_t pcm_size,
                               int sample_rate, int bit_depth, int channels,
                               encode_result *result) {
    memset(result, 0, sizeof(*result));

    UInt32 bytesPerSample = (UInt32)bit_depth / 8;

    // Source format: interleaved signed LE PCM.
    AudioStreamBasicDescription srcFormat;
    memset(&srcFormat, 0, sizeof(srcFormat));
    srcFormat.mSampleRate       = (Float64)sample_rate;
    srcFormat.mFormatID         = kAudioFormatLinearPCM;
    srcFormat.mFormatFlags      = kAudioFormatFlagIsSignedInteger | kAudioFormatFlagIsPacked;
    srcFormat.mBitsPerChannel   = (UInt32)bit_depth;
    srcFormat.mChannelsPerFrame = (UInt32)channels;
    srcFormat.mBytesPerFrame    = bytesPerSample * (UInt32)channels;
    srcFormat.mFramesPerPacket  = 1;
    srcFormat.mBytesPerPacket   = srcFormat.mBytesPerFrame;

    // Destination format: ALAC.
    AudioStreamBasicDescription dstFormat;
    memset(&dstFormat, 0, sizeof(dstFormat));
    dstFormat.mSampleRate       = (Float64)sample_rate;
    dstFormat.mFormatID         = kAudioFormatAppleLossless;
    dstFormat.mChannelsPerFrame = (UInt32)channels;
    dstFormat.mBitsPerChannel   = (UInt32)bit_depth;

    // Create temp file for encoding (AudioToolbox requires file URL).
    char tmpPath[] = "/tmp/coreaudio_encode_XXXXXX.m4a";
    int fd = mkstemps(tmpPath, 4);
    if (fd < 0) {
        snprintf(result->error, sizeof(result->error), "cannot create temp file");
        return 1;
    }
    close(fd);

    CFStringRef pathStr = CFStringCreateWithCString(kCFAllocatorDefault, tmpPath, kCFStringEncodingUTF8);
    if (!pathStr) {
        unlink(tmpPath);
        snprintf(result->error, sizeof(result->error), "cannot create path string");
        return 1;
    }

    CFURLRef outputURL = CFURLCreateWithFileSystemPath(kCFAllocatorDefault, pathStr, kCFURLPOSIXPathStyle, false);
    CFRelease(pathStr);
    if (!outputURL) {
        unlink(tmpPath);
        snprintf(result->error, sizeof(result->error), "cannot create output URL");
        return 1;
    }

    ExtAudioFileRef extFile = NULL;
    OSStatus status = ExtAudioFileCreateWithURL(
        outputURL, kAudioFileM4AType, &dstFormat, NULL,
        kAudioFileFlags_EraseFile, &extFile);
    CFRelease(outputURL);
    if (status != noErr) {
        unlink(tmpPath);
        snprintf(result->error, sizeof(result->error),
            "ExtAudioFileCreateWithURL failed (OSStatus %d)", (int)status);
        return 1;
    }

    status = ExtAudioFileSetProperty(extFile, kExtAudioFileProperty_ClientDataFormat,
        sizeof(srcFormat), &srcFormat);
    if (status != noErr) {
        ExtAudioFileDispose(extFile);
        unlink(tmpPath);
        snprintf(result->error, sizeof(result->error),
            "cannot set client format (OSStatus %d)", (int)status);
        return 1;
    }

    // Encode loop.
    const UInt32 framesPerWrite = 4096;
    int64_t totalFrames = pcm_size / srcFormat.mBytesPerFrame;
    int64_t framesWritten = 0;

    while (framesWritten < totalFrames) {
        int64_t remaining = totalFrames - framesWritten;
        UInt32 frameCount = (remaining < framesPerWrite) ? (UInt32)remaining : framesPerWrite;

        AudioBufferList bufList;
        bufList.mNumberBuffers = 1;
        bufList.mBuffers[0].mNumberChannels = (UInt32)channels;
        bufList.mBuffers[0].mDataByteSize   = frameCount * srcFormat.mBytesPerFrame;
        bufList.mBuffers[0].mData           = (void *)(pcm + framesWritten * srcFormat.mBytesPerFrame);

        status = ExtAudioFileWrite(extFile, frameCount, &bufList);
        if (status != noErr) {
            ExtAudioFileDispose(extFile);
            unlink(tmpPath);
            snprintf(result->error, sizeof(result->error),
                "ExtAudioFileWrite failed (OSStatus %d)", (int)status);
            return 1;
        }
        framesWritten += frameCount;
    }

    ExtAudioFileDispose(extFile);

    // Read encoded file back into memory.
    FILE *f = fopen(tmpPath, "rb");
    if (!f) {
        unlink(tmpPath);
        snprintf(result->error, sizeof(result->error), "cannot read encoded file");
        return 1;
    }

    fseek(f, 0, SEEK_END);
    long fileSize = ftell(f);
    fseek(f, 0, SEEK_SET);

    char *m4a = (char *)malloc(fileSize);
    if (!m4a) {
        fclose(f);
        unlink(tmpPath);
        snprintf(result->error, sizeof(result->error), "out of memory (%ld bytes)", fileSize);
        return 1;
    }

    size_t bytesRead = fread(m4a, 1, fileSize, f);
    fclose(f);
    unlink(tmpPath);

    if ((long)bytesRead != fileSize) {
        free(m4a);
        snprintf(result->error, sizeof(result->error), "read error");
        return 1;
    }

    result->m4a = m4a;
    result->m4a_size = fileSize;
    return 0;
   }
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// cgoCodec implements Codec using CGO AudioToolbox.
type cgoCodec struct{}

// NewCGO returns a Codec that uses CGO AudioToolbox (darwin only).
func NewCGO() Codec {
	return &cgoCodec{}
}

// Available reports whether CGO AudioToolbox is available.
func (c *cgoCodec) Available() bool {
	return true
}

// Decode decodes audio from memory using AudioToolbox.
func (c *cgoCodec) Decode(data []byte) ([]byte, Format, error) {
	if len(data) == 0 {
		return nil, Format{}, fmt.Errorf("%w: empty input", ErrDecodeFailed)
	}

	var result C.decode_result

	rc := C.coreaudio_decode(
		(*C.char)(unsafe.Pointer(&data[0])),
		C.int64_t(len(data)),
		&result,
	)

	if rc != 0 {
		return nil, Format{}, fmt.Errorf("%w: %s", ErrDecodeFailed, C.GoString(&result.error[0]))
	}

	pcm := C.GoBytes(unsafe.Pointer(result.pcm), C.int(result.pcm_size))
	C.free(unsafe.Pointer(result.pcm))

	format := Format{
		SampleRate: int(result.sample_rate),
		BitDepth:   int(result.bit_depth),
		Channels:   int(result.channels),
	}

	return pcm, format, nil
}

// Encode encodes raw PCM to ALAC M4A using AudioToolbox.
func (c *cgoCodec) Encode(pcm []byte, format Format) ([]byte, error) {
	if len(pcm) == 0 {
		return nil, fmt.Errorf("%w: empty input", ErrEncodeFailed)
	}

	if format.SampleRate <= 0 || format.BitDepth <= 0 || format.Channels <= 0 {
		return nil, fmt.Errorf("%w: invalid format", ErrEncodeFailed)
	}

	var result C.encode_result

	rc := C.coreaudio_encode(
		(*C.char)(unsafe.Pointer(&pcm[0])),
		C.int64_t(len(pcm)),
		C.int(format.SampleRate),
		C.int(format.BitDepth),
		C.int(format.Channels),
		&result,
	)

	if rc != 0 {
		return nil, fmt.Errorf("%w: %s", ErrEncodeFailed, C.GoString(&result.error[0]))
	}

	m4a := C.GoBytes(unsafe.Pointer(result.m4a), C.int(result.m4a_size))
	C.free(unsafe.Pointer(result.m4a))

	return m4a, nil
}
