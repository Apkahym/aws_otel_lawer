package invoke

import (
	"fmt"
	"os"
	"runtime/debug"
)

// RecoverPanic recupera de un panic y registra información de debug
func RecoverPanic() interface{} {
	if r := recover(); r != nil {
		stack := debug.Stack()
		fmt.Fprintf(os.Stderr, "PANIC recovered in wrapper:\n%v\n\nStack trace:\n%s\n", r, stack)
		return r
	}
	return nil
}

// SafeExecute ejecuta una función con recuperación de panic
func SafeExecute(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			stack := debug.Stack()
			fmt.Fprintf(os.Stderr, "Panic in SafeExecute:\n%v\n%s\n", r, stack)
		}
	}()

	return fn()
}
