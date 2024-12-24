package cmd

import (
	"context"
	"fmt"
	"main/manager"
	"main/run"
	"os"
)

// 在这里通过 init 函数自动注册命令
func init() {
	run.RegisterCommand(run.Command{
		Name:        "exit",
		Description: "Exits the interactive CLI",
		Action:      exitAction,
	})
}

func exitAction(ctx context.Context, params map[string]string) error {
	fmt.Println("Exiting the CLI...")

	manager.GetGRPCClient().Close()
	manager.GetDBManager().CloseDB()
	ctx.Done()

	os.Exit(0)
	return nil
}
