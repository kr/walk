package walk_test

import (
	"fmt"
	"os"

	"kr.dev/walk"
)

func ExampleWalker() {
	walker := walk.New(os.DirFS("/"), "usr/lib")
	for walker.Next() {
		if err := walker.Err(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Println(walker.Path())
	}
}
