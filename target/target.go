package target1

func callsPanic() {
	panic("error")
}

func noRecover() {
    callsPanic()
}

func withRecover() {
    defer recover()
    panic("error")
}