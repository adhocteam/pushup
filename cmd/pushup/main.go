package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/adhocteam/pushup/internal/command"
	"github.com/adhocteam/pushup/internal/compile"
)

type subcmd struct {
	name  string
	setup func(*flag.FlagSet)
	run   func(*flag.FlagSet) error
}

var subcommands = []subcmd{
	{
		name: "build",
		setup: func(fs *flag.FlagSet) {
			fs.String("r", ".", "Build project from `root` directory")
		},
		run: func(fs *flag.FlagSet) error {
			root := fs.Lookup("r").Value.String()
			return command.Build(root)
		},
	},
	{
		name: "compile",
		setup: func(fs *flag.FlagSet) {
			fs.Bool("print-ast", false, "Pretty-print the AST and then exit")
		},
		run: func(fs *flag.FlagSet) error {
			if fs.NArg() < 1 {
				return fmt.Errorf("missing file argument")
			}
			prettyPrint := fs.Lookup("print-ast").Value.(flag.Getter).Get().(bool)
			filename := fs.Arg(0)
			if prettyPrint {
				return command.PrettyPrintAST(filename)
			}
			_, err := compile.Compile(filename)
			return err
		},
	},
}

func main() {
	flag.Usage = printUsage

	flag.Parse()

	if !findGo() {
		fmt.Fprintf(os.Stderr, "Pushup requires Go.\n")
		os.Exit(1)
	}

	if len(flag.Args()) < 1 {
		printUsage()
		os.Exit(1)
	}

	cmdName := flag.Arg(0)
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

	err := fs.Parse(flag.Args()[1:])
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

func findCommand(name string) *subcmd {
	for i := range subcommands {
		if subcommands[i].name == name {
			return &subcommands[i]
		}
	}
	return nil
}

func printUsage() {
	fmt.Fprintln(flag.CommandLine.Output(), "Usage: pushup <command>")
	fmt.Fprintln(flag.CommandLine.Output(), "Commands:")
	for _, cmd := range subcommands {
		fmt.Fprintf(flag.CommandLine.Output(), "  %s\n", cmd.name)
	}
}

func findGo() bool {
	_, err := exec.LookPath("go")
	if err != nil {
		return false
	}
	return true
}
