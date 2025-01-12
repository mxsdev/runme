package cmd

import (
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type program struct {
	*tea.Program
	out io.Writer
}

func (p *program) Start() error {
	if f, ok := p.out.(*os.File); ok && !isTerminal(f.Fd()) {
		go p.Quit()
	}
	_, err := p.Program.Run()
	return err
}

func newProgram(cmd *cobra.Command, model tea.Model) *program {
	out := cmd.OutOrStdout()
	return &program{
		Program: tea.NewProgram(
			model,
			tea.WithOutput(out),
			tea.WithInput(cmd.InOrStdin()),
		),
		out: out,
	}
}
