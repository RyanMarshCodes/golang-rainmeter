//go:build windows && cgo

package winrt

// #include "c/helpers.h"
import "C"
import "unsafe"

// GetActivationFactory returns the activation factory COM interface pointer
// for a WinRT runtime class. The returned pointer is an IInspectable* or
// specific statics interface pointer depending on the iid parameter.
//
// The caller is responsible for calling Release() on the returned pointer.
func GetActivationFactory(className string, iid *GUID) (unsafe.Pointer, error) {
	hstr, err := NewHSTRING(className)
	if err != nil {
		return nil, err
	}
	defer hstr.Delete()

	var factory unsafe.Pointer
	hr := C.RoGetActivationFactory(
		hstr.Raw(),
		(*C.IID)(unsafe.Pointer(iid)),
		&factory,
	)
	if hr < 0 {
		return nil, hresultError("RoGetActivationFactory("+className+")", hr)
	}
	return factory, nil
}

// ActivateInstance activates a WinRT runtime class and returns its default
// interface as an IInspectable pointer.
//
// The caller is responsible for calling Release() on the returned pointer.
func ActivateInstance(className string) (unsafe.Pointer, error) {
	hstr, err := NewHSTRING(className)
	if err != nil {
		return nil, err
	}
	defer hstr.Delete()

	var instance unsafe.Pointer
	hr := C.RoActivateInstance(
		hstr.Raw(),
		&instance,
	)
	if hr < 0 {
		return nil, hresultError("RoActivateInstance("+className+")", hr)
	}
	return instance, nil
}
