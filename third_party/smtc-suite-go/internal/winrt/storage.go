//go:build windows && cgo

package winrt

import (
	"fmt"
	"net/url"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	runtimeClassStorageFile                 = "Windows.Storage.StorageFile"
	runtimeClassRandomAccessStreamReference = "Windows.Storage.Streams.RandomAccessStreamReference"
	runtimeClassUri                         = "Windows.Foundation.Uri"

	slotStorageFileStaticsGetFileFromPathAsync = 6
	slotStreamReferenceStaticsCreateFromFile   = 6
	slotStreamReferenceStaticsCreateFromUri    = 7
	slotUriFactoryCreateUri                    = 6
)

// RandomAccessStreamReferenceFromFile creates an IRandomAccessStreamReference
// for a local file path.
//
// The returned pointer is owned by the caller and must be released with
// Release. The path must be an absolute or Windows-resolvable filesystem path
// that StorageFile.GetFileFromPathAsync can open.
func RandomAccessStreamReferenceFromFile(path string) (unsafe.Pointer, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	file, err := storageFileFromPath(abs)
	if err == nil {
		defer Release(file)
		return randomAccessStreamReferenceFromStorageFile(file)
	}

	if ref, uriErr := randomAccessStreamReferenceFromURI(fileURI(abs)); uriErr == nil {
		return ref, nil
	}
	return nil, err
}

func storageFileFromPath(path string) (unsafe.Pointer, error) {
	factory, err := GetActivationFactory(runtimeClassStorageFile, IID_IStorageFileStatics)
	if err != nil {
		return nil, fmt.Errorf("storage file factory: %w", err)
	}
	defer Release(factory)

	hstr, err := NewHSTRING(path)
	if err != nil {
		return nil, err
	}
	defer hstr.Delete()

	fn := vtableFn(factory, slotStorageFileStaticsGetFileFromPathAsync)
	var asyncPtr unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(factory),
		uintptr(unsafe.Pointer(hstr.Raw())),
		uintptr(unsafe.Pointer(&asyncPtr)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("StorageFile.GetFileFromPathAsync", int32(r1))
	}

	asyncOp := NewAsyncOperation(asyncPtr)
	defer asyncOp.Release()
	file, err := asyncOp.Wait()
	if err != nil {
		return nil, fmt.Errorf("wait for StorageFile: %w", err)
	}
	return file, nil
}

// StorageFileFromPath returns a Windows.Storage.StorageFile for a local path.
//
// The returned pointer is owned by the caller and must be released with
// Release.
func StorageFileFromPath(path string) (unsafe.Pointer, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return storageFileFromPath(abs)
}

// RandomAccessStreamReferenceFromURI creates an IRandomAccessStreamReference
// from an absolute URI such as https://example.com/cover.jpg or file:///C:/cover.jpg.
//
// The returned pointer is owned by the caller and must be released with Release.
func RandomAccessStreamReferenceFromURI(uri string) (unsafe.Pointer, error) {
	return randomAccessStreamReferenceFromURI(uri)
}

func randomAccessStreamReferenceFromStorageFile(file unsafe.Pointer) (unsafe.Pointer, error) {
	factory, err := GetActivationFactory(runtimeClassRandomAccessStreamReference, IID_IRandomAccessStreamReferenceStatics)
	if err != nil {
		return nil, fmt.Errorf("stream reference factory: %w", err)
	}
	defer Release(factory)

	fn := vtableFn(factory, slotStreamReferenceStaticsCreateFromFile)
	var ref unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(factory),
		uintptr(file),
		uintptr(unsafe.Pointer(&ref)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("RandomAccessStreamReference.CreateFromFile", int32(r1))
	}
	return ref, nil
}

func randomAccessStreamReferenceFromURI(uriString string) (unsafe.Pointer, error) {
	uri, err := uriFromString(uriString)
	if err != nil {
		return nil, err
	}
	defer Release(uri)

	factory, err := GetActivationFactory(runtimeClassRandomAccessStreamReference, IID_IRandomAccessStreamReferenceStatics)
	if err != nil {
		return nil, fmt.Errorf("stream reference factory: %w", err)
	}
	defer Release(factory)

	fn := vtableFn(factory, slotStreamReferenceStaticsCreateFromUri)
	var ref unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(factory),
		uintptr(uri),
		uintptr(unsafe.Pointer(&ref)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("RandomAccessStreamReference.CreateFromUri", int32(r1))
	}
	return ref, nil
}

func uriFromString(raw string) (unsafe.Pointer, error) {
	factory, err := GetActivationFactory(runtimeClassUri, IID_IUriRuntimeClassFactory)
	if err != nil {
		return nil, fmt.Errorf("uri factory: %w", err)
	}
	defer Release(factory)

	hstr, err := NewHSTRING(raw)
	if err != nil {
		return nil, err
	}
	defer hstr.Delete()

	fn := vtableFn(factory, slotUriFactoryCreateUri)
	var uri unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(factory),
		uintptr(unsafe.Pointer(hstr.Raw())),
		uintptr(unsafe.Pointer(&uri)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("Uri.CreateUri", int32(r1))
	}
	return uri, nil
}

func FileURI(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return fileURI(abs), nil
}

func fileURI(path string) string {
	u := url.URL{
		Scheme: "file",
		Path:   "/" + filepath.ToSlash(path),
	}
	return u.String()
}
