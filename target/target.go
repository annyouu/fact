package target

import "fmt"

func sayHello() {
	fmt.Println("Hello")
	// return文なし
}

func sayBye() string {
	fmt.Println("Bye")
	return "ok" // return あり
}

func main() {
	sayHello() // 警告を出す
	sayBye() // 警告を出さない
}

