package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/LegendLoreLori/radgregator/internal/config"
)

type state struct {
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	cmds map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.cmds[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	f, ok := c.cmds[cmd.name]
	if !ok {
		return fmt.Errorf("command: %q not found", cmd.name)
	}
	if err := f(s, cmd); err != nil {
		return err
	}
	return nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return errors.New("missing positional argument [USERNAME]")
	}

	if err := s.cfg.SetUser(cmd.args[1]); err != nil {
		return fmt.Errorf("error calling config method SetUser: %w", err)
	}
	println("username set")
	return nil
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	s := state{&cfg}
	c := commands{make(map[string]func(*state, command) error)}

	c.register("login", handlerLogin)

	if len(os.Args) < 2 {
		fmt.Println("missing positional arguments, I'll implement a better way to handle this later")
		os.Exit(1)
	}
	cmd := command{os.Args[1], os.Args[1:]}
	if err := c.run(&s, cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
