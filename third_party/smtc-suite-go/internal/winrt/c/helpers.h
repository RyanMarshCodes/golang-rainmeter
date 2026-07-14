// +build windows

// Package winrt/c provides C helper declarations for WinRT COM interop.
// This header is included by Go files via CGo preambles.
//
// Since MinGW-w64 does not ship WinRT headers (roapi.h, hstring.h),
// we declare all needed types and functions manually here.

#ifndef SMTC_WINRT_HELPERS_H
#define SMTC_WINRT_HELPERS_H

#include <windows.h>
#include <ole2.h>
#include <stdint.h>

// ---- HSTRING (opaque handle, defined in <hstring.h> by Windows SDK) ----
typedef struct HSTRING__ { int unused; } *HSTRING;

// ---- HSTRING_HEADER (for WindowsCreateStringReference) ----
typedef struct HSTRING_HEADER {
    union {
        PVOID Reserved1;
        char Reserved2[24];
    };
} HSTRING_HEADER;

// ---- RO_INIT_TYPE (from <roapi.h>) ----
typedef enum {
    RO_INIT_SINGLETHREADED = 0,
    RO_INIT_MULTITHREADED  = 1
} RO_INIT_TYPE;

// ---- EventRegistrationToken (from <eventtoken.h>) ----
typedef struct {
    int64_t value;
} EventRegistrationToken;

// ---- TrustLevel (from <inspectable.h>) ----
typedef enum {
    BaseTrust = 0,
    PartialTrust = 1,
    FullTrust = 2
} TrustLevel;

// ============================================================
// WinRT Functions imported from runtimeobject.dll / combase.dll
// ============================================================

// ---- RoInitialize / RoUninitialize (runtimeobject.dll) ----
HRESULT WINAPI RoInitialize(RO_INIT_TYPE initType);
void WINAPI RoUninitialize(void);

// ---- RoGetActivationFactory (runtimeobject.dll) ----
HRESULT WINAPI RoGetActivationFactory(
    HSTRING activatableClassId,
    REFIID  iid,
    void    **factory
);

HRESULT WINAPI RoActivateInstance(
    HSTRING activatableClassId,
    void **instance
);

// ---- HSTRING management (combase.dll) ----
HRESULT WINAPI WindowsCreateString(
    LPCWSTR  sourceString,
    UINT32   length,
    HSTRING  *string
);

HRESULT WINAPI WindowsDeleteString(
    HSTRING string
);

HRESULT WINAPI WindowsCreateStringReference(
    LPCWSTR        sourceString,
    UINT32         length,
    HSTRING_HEADER *hstringHeader,
    HSTRING        *string
);

LPCWSTR WINAPI WindowsGetStringRawBuffer(
    HSTRING string,
    UINT32 *length
);

// ============================================================
// MinGW compatibility: IInspectable interface
// ============================================================

#ifndef __IInspectable_INTERFACE_DEFINED__
#define __IInspectable_INTERFACE_DEFINED__

// IInspectable IID: {AF86E2E0-B12D-4C6A-9C5A-D7AA65101E90}
static const IID IID_IInspectable = {
    0xAF86E2E0, 0xB12D, 0x4C6A,
    {0x9C, 0x5A, 0xD7, 0xAA, 0x65, 0x10, 0x1E, 0x90}
};

typedef struct IInspectable IInspectable;
typedef struct IInspectableVtbl IInspectableVtbl;

struct IInspectableVtbl {
    // IUnknown
    HRESULT (STDMETHODCALLTYPE *QueryInterface)(IInspectable *This, REFIID riid, void **ppvObject);
    ULONG   (STDMETHODCALLTYPE *AddRef)(IInspectable *This);
    ULONG   (STDMETHODCALLTYPE *Release)(IInspectable *This);
    // IInspectable
    HRESULT (STDMETHODCALLTYPE *GetIids)(IInspectable *This, ULONG *iidCount, IID **iids);
    HRESULT (STDMETHODCALLTYPE *GetRuntimeClassName)(IInspectable *This, HSTRING *className);
    HRESULT (STDMETHODCALLTYPE *GetTrustLevel)(IInspectable *This, TrustLevel *trustLevel);
};

struct IInspectable {
    IInspectableVtbl *lpVtbl;
};

#endif // __IInspectable_INTERFACE_DEFINED__

// ============================================================
// Helpers for CGo event/async bridging
// ============================================================

// Function pointer type for Go callbacks invoked from WinRT event handlers.
typedef void (*GoEventCallback)(void *sender, void *args, void *userData);

typedef HRESULT (STDMETHODCALLTYPE *SMTCVtablePutF64Fn)(void *self, double value);
typedef HRESULT (STDMETHODCALLTYPE *SMTCAsyncF64Fn)(void *self, double value, void **operation);

static inline HRESULT smtcVtablePutF64(void *self, void *fn, double value) {
    return ((SMTCVtablePutF64Fn)fn)(self, value);
}

static inline HRESULT smtcAsyncF64(void *self, void *fn, double value, void **operation) {
    return ((SMTCAsyncF64Fn)fn)(self, value, operation);
}

#endif // SMTC_WINRT_HELPERS_H
