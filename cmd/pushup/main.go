package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/adhocteam/pushup/internal"
)

type command struct {
	name  string
	setup func(*flag.FlagSet)
	run   func(*flag.FlagSet) error
}

var commands = []command{
	{
		name: "build",
		setup: func(fs *flag.FlagSet) {
			fs.String("root", ".", "Root directory")
		},
		run: func(fs *flag.FlagSet) error {
			root := fs.Lookup("root").Value.String()
			return internal.Build(root)
		},
	},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: pushup <command>")
		os.Exit(1)
	}

	cmdName := os.Args[1]
	cmd := findCommand(cmdName)
	if cmd == nil {
		fmt.Printf("Unknown command: %s\n", cmdName)
		printUsage()
		os.Exit(1)
	}

	fs := flag.NewFlagSet(cmdName, flag.ExitOnError)
	cmd.setup(fs)
	fs.Usage = func() {
		fmt.Printf("Usage: pushup %s [flags]\n", cmdName)
		fs.PrintDefaults()
	}

	err := fs.Parse(os.Args[2:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = cmd.run(fs)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func findCommand(name string) *command {
	for i := range commands {
		if commands[i].name == name {
			return &commands[i]
		}
	}
	return nil
}

func printUsage() {
	fmt.Println("Usage: pushup <command>")
	fmt.Println("Commands:")
	for _, cmd := range commands {
		fmt.Printf("  %s\n", cmd.name)
	}
}
