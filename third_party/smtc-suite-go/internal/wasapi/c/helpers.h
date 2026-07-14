// +build windows

#ifndef SMTC_WASAPI_HELPERS_H
#define SMTC_WASAPI_HELPERS_H

#include <windows.h>
#include <ole2.h>
#include <mmsystem.h>
#include <mmreg.h>
#include <ksmedia.h>
#include <stdint.h>

#ifndef WAVE_FORMAT_PCM
#define WAVE_FORMAT_PCM 0x0001
#endif

#ifndef WAVE_FORMAT_IEEE_FLOAT
#define WAVE_FORMAT_IEEE_FLOAT 0x0003
#endif

#ifndef WAVE_FORMAT_EXTENSIBLE
#define WAVE_FORMAT_EXTENSIBLE 0xFFFE
#endif

static inline WORD smtcWaveFormatTag(void *p) {
    return ((WAVEFORMATEX*)p)->wFormatTag;
}

static inline WORD smtcWaveFormatChannels(void *p) {
    return ((WAVEFORMATEX*)p)->nChannels;
}

static inline DWORD smtcWaveFormatSamplesPerSec(void *p) {
    return ((WAVEFORMATEX*)p)->nSamplesPerSec;
}

static inline DWORD smtcWaveFormatAvgBytesPerSec(void *p) {
    return ((WAVEFORMATEX*)p)->nAvgBytesPerSec;
}

static inline WORD smtcWaveFormatBlockAlign(void *p) {
    return ((WAVEFORMATEX*)p)->nBlockAlign;
}

static inline WORD smtcWaveFormatBitsPerSample(void *p) {
    return ((WAVEFORMATEX*)p)->wBitsPerSample;
}

static inline WORD smtcWaveFormatCbSize(void *p) {
    return ((WAVEFORMATEX*)p)->cbSize;
}

static inline WORD smtcWaveFormatValidBitsPerSample(void *p) {
    return ((WAVEFORMATEXTENSIBLE*)p)->Samples.wValidBitsPerSample;
}

static inline DWORD smtcWaveFormatChannelMask(void *p) {
    return ((WAVEFORMATEXTENSIBLE*)p)->dwChannelMask;
}

static inline GUID* smtcWaveFormatSubFormat(void *p) {
    return &((WAVEFORMATEXTENSIBLE*)p)->SubFormat;
}

#endif // SMTC_WASAPI_HELPERS_H
