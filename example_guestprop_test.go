package virtualbox_test

import (
	"fmt"
	"log"
	"sync"
	"time"

	virtualbox "github.com/terra-farm/go-virtualbox"
)

var VM = "MyVM"

func ExampleSetGuestProperty() {
	err := virtualbox.SetGuestProperty(VM, "test_name", "test_val")
	if err != nil {
		panic(err)
	}
}

func ExampleGetGuestProperty() {
	err := virtualbox.SetGuestProperty(VM, "test_name", "test_val")
	if err != nil {
		panic(err)
	}
	val, err := virtualbox.GetGuestProperty(VM, "test_name")
	if err != nil {
		panic(err)
	}
	log.Println("val:", val)
}

func ExampleDeleteGuestProperty() {
	err := virtualbox.SetGuestProperty(VM, "test_name", "test_val")
	if err != nil {
		panic(err)
	}
	err = virtualbox.DeleteGuestProperty(VM, "test_name")
	if err != nil {
		panic(err)
	}
}

func ExampleWaitGuestProperty() {

	go func() {
		second := time.Second
		time.Sleep(1 * second)
		err := virtualbox.SetGuestProperty(VM, "test_name", "test_val")
		onErrPanic(err, "failed to SetGuestProperty(VM, test_name, test_val)")
	}()

	name, val, err := virtualbox.WaitGuestProperty(VM, "test_*")
	if err != nil {
		panic(err)
	}
	log.Println("name:", name, ", value:", val)
}

func ExampleWaitGuestProperties() {
	go func() {
		second := time.Second

		time.Sleep(1 * second)
		err := virtualbox.SetGuestProperty(VM, "test_name", "test_val1")

		onErrPanic(err, ">>> failed to set guest property key='test_name', val='test_val1', err=%v", err)

		time.Sleep(1 * second)
		err = virtualbox.SetGuestProperty(VM, "test_name", "test_val2")
		onErrPanic(err, ">>> failed to set guest property key='test_name', val='test_val2', err=%v", err)

		time.Sleep(1 * second)
		err = virtualbox.SetGuestProperty(VM, "test_name", "test_val1")
		onErrPanic(err, ">>> failed to set guest property key='test_name', val='test_val1', err=%v", err)
	}()

	wg := new(sync.WaitGroup)
	done := make(chan bool)
	propsPattern := "test_*"
	props := virtualbox.WaitGuestProperties(VM, propsPattern, done, wg)

	ok := true
	left := 3
	for ; ok && left > 0; left-- {
		var prop virtualbox.GuestProperty
		prop, ok = <-props
		log.Println("name:", prop.Name, ", value:", prop.Value)
	}

	close(done) // close channel
	wg.Wait()   // wait for gorouting
}

func onErrPanic(err error, msg string, args ...interface{}) {
	if err == nil {
		return
	}
	panic(fmt.Sprintf(msg, args...))
}
