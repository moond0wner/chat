package core_command

import (
	"fmt"
	"strings"
	core_domain "tcp_srv/internal/core/domain"
)

type CommandHandler struct {
	Name        string
	Description string
	Usage       string
	MinArgs     int
	Handler     func(client *core_domain.Client, args []string) error
}

func (c CommandHandler) Execute(client *core_domain.Client, args []string) error {
	if len(args) < c.MinArgs {
		return fmt.Errorf("Usage: %s", c.Usage)
	}
	return c.Handler(client, args)
}

type Manager struct {
	commands map[string]CommandHandler
}

func NewManager() *Manager {
	return &Manager{
		commands: make(map[string]CommandHandler),
	}
}

func (m *Manager) Register(cmds []CommandHandler) {
	for _, cmd := range cmds {
		m.commands[cmd.Name] = cmd
	}
}

func (m *Manager) Get(name string) (CommandHandler, bool) {
	cmd, ok := m.commands[name]
	return cmd, ok
}

func (m *Manager) List() []CommandHandler {
	cmds := make([]CommandHandler, 0, len(m.commands))
	for _, cmd := range m.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

func (m *Manager) Execute(input string, client *core_domain.Client) error {
	parts := strings.SplitN(input, " ", 2)
	name := parts[0]
	args := []string{}
	if len(parts) > 1 {
		args = strings.Fields(parts[1])
	}

	cmd, ok := m.Get(name)
	if !ok {
		return fmt.Errorf("unknown command: %s", name)
	}

	return cmd.Execute(client, args)
}
