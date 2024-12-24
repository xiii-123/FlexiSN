package cmd

import (
	"context"
	"fmt"
	"main/run"
)

// 在这里通过 init 函数自动注册命令
func init() {
	run.RegisterCommand(run.Command{
		Name:        "hello",
		Description: "Prints a welcome message",
		Action:      helloAction,
	})
}

func helloAction(ctx context.Context, params map[string]string) error {
	if name, exists := params["-p"]; exists {
		fmt.Printf("Hello, %s! Welcome to the interactive CLI!\n", name)
	}
	fmt.Println("Hello, welcome to the interactive CLI!")
	return nil
}
