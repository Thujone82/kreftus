//go:build windows || darwin || ((linux || freebsd || netbsd || openbsd) && !android && !nintendosdk && !playstation5)

package noinit

// Imports must stay a subset of what Ebitengine's internal/ui reaches so the
// linker schedules this package's inittask before ui's (tie-break by path:
// github.com/bstkhq/... < github.com/hajimehoshi/...).
import (
	"image"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"unsafe"
)

type initTask struct {
	state uint32
	nfns  uint32
}

//go:linkname uiInitTask github.com/hajimehoshi/ebiten/v2/internal/ui..inittask
var uiInitTask initTask

//go:linkname uiInit github.com/hajimehoshi/ebiten/v2/internal/ui.init.0
func uiInit()

//go:linkname theUI github.com/hajimehoshi/ebiten/v2/internal/ui.theUI
var theUI unsafe.Pointer

//go:linkname uiSetScreenClearedEveryFrame github.com/hajimehoshi/ebiten/v2/internal/ui.(*UserInterface).SetScreenClearedEveryFrame
func uiSetScreenClearedEveryFrame(u unsafe.Pointer, cleared bool)

//go:linkname uiNewImage github.com/hajimehoshi/ebiten/v2/internal/ui.(*UserInterface).NewImage
func uiNewImage(u unsafe.Pointer, width, height, imageType int) unsafe.Pointer

//go:linkname uiImageWritePixels github.com/hajimehoshi/ebiten/v2/internal/ui.(*Image).WritePixels
func uiImageWritePixels(i unsafe.Pointer, pix []byte, region image.Rectangle)

//go:linkname typesByString reflect.typesByString
func typesByString(string) []unsafe.Pointer

func init() {
	if !shouldSkipEbitenUI() {
		return
	}
	if uiInitTask.state != 0 {
		return
	}

	target := functionPC(uiInit)
	verifyUIInitializer(target)

	if !replaceInitializer(&uiInitTask, target, functionPC(initializeWithoutUI)) {
		panic("ebitennoinit: Ebitengine's UI initializer was not found; unsupported Ebitengine or Go version")
	}
}

const uiInitFile = "/internal/ui/ui.go"

func verifyUIInitializer(pc uintptr) {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		panic("ebitennoinit: Ebitengine's UI initializer has no symbol information")
	}
	if file, _ := fn.FileLine(pc); !strings.HasSuffix(file, uiInitFile) {
		panic("ebitennoinit: " + fn.Name() + " is declared in " + file + ", not " + uiInitFile +
			"; unsupported Ebitengine version")
	}
}

const maxInitializers = 1024

func replaceInitializer(task *initTask, target, replacement uintptr) bool {
	if task.nfns == 0 || task.nfns > maxInitializers {
		return false
	}

	first := unsafe.Add(unsafe.Pointer(task), unsafe.Sizeof(*task))
	var match *uintptr
	for i := uint32(0); i < task.nfns; i++ {
		slot := (*uintptr)(unsafe.Add(first, uintptr(i)*unsafe.Sizeof(uintptr(0))))
		if *slot == target {
			if match != nil {
				return false
			}
			match = slot
		}
	}
	if match == nil {
		return false
	}
	*match = replacement
	return true
}

func functionPC(fn func()) uintptr {
	return reflect.ValueOf(fn).Pointer()
}

//go:noinline
func initializeWithoutUI() {
	u := reflect.New(userInterfaceType())
	uPointer := u.UnsafePointer()

	uiSetScreenClearedEveryFrame(uPointer, true)
	if atomicWordOf(u.Elem(), "isScreenClearedEveryFrame").IsZero() {
		panic("ebitennoinit: Ebitengine SetScreenClearedEveryFrame had no effect; unsupported Ebitengine version")
	}

	graphicsLibrary := atomicWordOf(u.Elem(), "graphicsLibrary")
	if graphicsLibrary.Kind() != reflect.Int32 {
		panic("ebitennoinit: Ebitengine UserInterface.graphicsLibrary changed type")
	}
	atomic.StoreInt32((*int32)(unsafe.Pointer(graphicsLibrary.UnsafeAddr())), 1)

	whiteImage := uiNewImage(uPointer, 3, 3, 0)
	if whiteImage == nil {
		panic("ebitennoinit: Ebitengine NewImage returned nothing")
	}

	whiteImageField := requiredField(u.Elem(), "whiteImage")
	whiteImageValue := reflect.NewAt(whiteImageField.Type().Elem(), whiteImage)
	verifyImage(whiteImageValue.Elem(), uPointer)
	reflect.NewAt(whiteImageField.Type(), unsafe.Pointer(whiteImageField.UnsafeAddr())).Elem().Set(whiteImageValue)

	pix := make([]byte, 4*3*3)
	for i := range pix {
		pix[i] = 0xff
	}
	uiImageWritePixels(whiteImage, pix, image.Rect(0, 0, 3, 3))

	theUI = uPointer
	runtime.KeepAlive(u)
}

func verifyImage(img reflect.Value, owner unsafe.Pointer) {
	if requiredField(img, "width").Int() != 3 || requiredField(img, "height").Int() != 3 {
		panic("ebitennoinit: Ebitengine NewImage ignored its size; unsupported Ebitengine version")
	}
	if requiredField(img, "ui").UnsafePointer() != owner {
		panic("ebitennoinit: Ebitengine NewImage ignored its receiver; unsupported Ebitengine version")
	}
}

func atomicWordOf(owner reflect.Value, name string) reflect.Value {
	field := requiredField(owner, name)
	if field.Kind() != reflect.Struct {
		panic("ebitennoinit: Ebitengine UserInterface." + name + " is not a sync/atomic value")
	}

	var word reflect.Value
	for i := 0; i < field.NumField(); i++ {
		f := field.Field(i)
		if f.Type().Size() == 0 {
			continue
		}
		if word.IsValid() {
			panic("ebitennoinit: unexpected sync/atomic layout for " + name)
		}
		word = f
	}
	if !word.IsValid() {
		panic("ebitennoinit: unexpected sync/atomic layout for " + name)
	}
	return word
}

func requiredField(value reflect.Value, name string) reflect.Value {
	field := value.FieldByName(name)
	if !field.IsValid() || !field.CanAddr() {
		panic("ebitennoinit: Ebitengine field not found: " + name)
	}
	return field
}

func userInterfaceType() reflect.Type {
	var found reflect.Type
	for _, pointer := range typesByString("*ui.UserInterface") {
		t := asReflectType(pointer)
		if t.Kind() != reflect.Pointer {
			continue
		}
		elem := t.Elem()
		if elem.Name() != "UserInterface" || elem.PkgPath() != "github.com/hajimehoshi/ebiten/v2/internal/ui" {
			continue
		}
		if found != nil {
			panic("ebitennoinit: Ebitengine UserInterface type was not unique")
		}
		found = elem
	}
	if found == nil {
		panic("ebitennoinit: Ebitengine UserInterface type was not found")
	}
	return found
}

func asReflectType(pointer unsafe.Pointer) reflect.Type {
	template := reflect.TypeOf(0)
	words := *(*[2]unsafe.Pointer)(unsafe.Pointer(&template))
	words[1] = pointer
	return *(*reflect.Type)(unsafe.Pointer(&words))
}
