//go:build windows && cgo

package winrt

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	runtimeClassBuffer = "Windows.Storage.Streams.Buffer"

	slotRandomAccessStreamReferenceOpenReadAsync = 6
	slotRandomAccessStreamGetSize                = 6
	slotInputStreamReadAsync                     = 6
	slotBufferGetLength                          = 7
	slotBufferFactoryCreate                      = 6

	inputStreamOptionsPartial = 1
)

const maxThumbnailBytes = 16 * 1024 * 1024

// ReadRandomAccessStreamReference reads all bytes from an IRandomAccessStreamReference.
func ReadRandomAccessStreamReference(ref unsafe.Pointer) ([]byte, error) {
	if ref == nil {
		return nil, nil
	}

	refStream, err := QueryInterface(ref, IID_IRandomAccessStreamReference)
	if err != nil {
		return nil, fmt.Errorf("IRandomAccessStreamReference: %w", err)
	}
	defer Release(refStream)

	stream, err := openRandomAccessStreamReference(refStream)
	if err != nil {
		return nil, err
	}
	defer Release(stream)

	randomStream, err := QueryInterface(stream, IID_IRandomAccessStream)
	if err != nil {
		return nil, fmt.Errorf("IRandomAccessStream: %w", err)
	}
	defer Release(randomStream)

	inputStream, err := QueryInterface(randomStream, IID_IInputStream)
	if err != nil {
		return nil, fmt.Errorf("IInputStream: %w", err)
	}
	defer Release(inputStream)

	size, err := randomAccessStreamSize(randomStream)
	if err != nil {
		return nil, err
	}
	if size == 0 {
		return nil, nil
	}
	if size > maxThumbnailBytes {
		return nil, fmt.Errorf("thumbnail stream too large: %d bytes", size)
	}

	buffer, err := createBuffer(uint32(size))
	if err != nil {
		return nil, err
	}
	defer Release(buffer)

	filled, err := inputStreamRead(inputStream, buffer, uint32(size))
	if err != nil {
		return nil, err
	}
	defer Release(filled)

	length, err := bufferLength(filled)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, nil
	}

	data, err := bufferBytes(filled, length)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func openRandomAccessStreamReference(ref unsafe.Pointer) (unsafe.Pointer, error) {
	fn := vtableFn(ref, slotRandomAccessStreamReferenceOpenReadAsync)
	var asyncPtr unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(ref),
		uintptr(unsafe.Pointer(&asyncPtr)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("IRandomAccessStreamReference.OpenReadAsync", int32(r1))
	}

	asyncOp := NewAsyncOperation(asyncPtr)
	defer asyncOp.Release()
	stream, err := asyncOp.Wait()
	if err != nil {
		return nil, fmt.Errorf("wait for thumbnail stream: %w", err)
	}
	return stream, nil
}

func randomAccessStreamSize(stream unsafe.Pointer) (uint64, error) {
	fn := vtableFn(stream, slotRandomAccessStreamGetSize)
	var size uint64
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(stream),
		uintptr(unsafe.Pointer(&size)),
	)
	if int32(r1) < 0 {
		return 0, hresultErrorInt("IRandomAccessStream.get_Size", int32(r1))
	}
	return size, nil
}

func createBuffer(capacity uint32) (unsafe.Pointer, error) {
	factory, err := GetActivationFactory(runtimeClassBuffer, IID_IBufferFactory)
	if err != nil {
		return nil, fmt.Errorf("buffer factory: %w", err)
	}
	defer Release(factory)

	fn := vtableFn(factory, slotBufferFactoryCreate)
	var buffer unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(factory),
		uintptr(capacity),
		uintptr(unsafe.Pointer(&buffer)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("Buffer.Create", int32(r1))
	}
	return buffer, nil
}

func inputStreamRead(stream unsafe.Pointer, buffer unsafe.Pointer, count uint32) (unsafe.Pointer, error) {
	fn := vtableFn(stream, slotInputStreamReadAsync)
	var asyncPtr unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(stream),
		uintptr(buffer),
		uintptr(count),
		uintptr(inputStreamOptionsPartial),
		uintptr(unsafe.Pointer(&asyncPtr)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("IInputStream.ReadAsync", int32(r1))
	}
	if asyncPtr == nil {
		return nil, fmt.Errorf("IInputStream.ReadAsync returned nil operation")
	}

	asyncOp := NewAsyncOperationWithProgress(asyncPtr)
	defer asyncOp.Release()
	filled, err := asyncOp.Wait()
	if err != nil {
		return nil, fmt.Errorf("wait for thumbnail read: %w", err)
	}
	return filled, nil
}

func bufferLength(buffer unsafe.Pointer) (uint32, error) {
	fn := vtableFn(buffer, slotBufferGetLength)
	var length uint32
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(buffer),
		uintptr(unsafe.Pointer(&length)),
	)
	if int32(r1) < 0 {
		return 0, hresultErrorInt("IBuffer.get_Length", int32(r1))
	}
	return length, nil
}

func bufferBytes(buffer unsafe.Pointer, length uint32) ([]byte, error) {
	byteAccess, err := QueryInterface(buffer, IID_IBufferByteAccess)
	if err != nil {
		return nil, fmt.Errorf("IBufferByteAccess: %w", err)
	}
	defer Release(byteAccess)

	fn := vtableFn(byteAccess, 3)
	var dataPtr unsafe.Pointer
	r1, _, _ := syscall.SyscallN(fn,
		uintptr(byteAccess),
		uintptr(unsafe.Pointer(&dataPtr)),
	)
	if int32(r1) < 0 {
		return nil, hresultErrorInt("IBufferByteAccess.Buffer", int32(r1))
	}
	if dataPtr == nil {
		return nil, nil
	}

	return append([]byte(nil), unsafe.Slice((*byte)(dataPtr), int(length))...), nil
}
