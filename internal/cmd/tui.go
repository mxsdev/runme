package cmd

import (
	"fmt"
	"math"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	"github.com/stateful/runme/internal/cmd/util"
	"github.com/stateful/runme/internal/document"
)

type tuiModel struct {
	blocks     document.CodeBlocks
	cursor     *int
	expanded   map[int]struct{}
	version    string
	run        **document.CodeBlock
	numEntries int
	scroll     *int
}

func (m *tuiModel) numBlocksShown() int {
	return util.Min(len(m.blocks), m.numEntries)
}

func (m *tuiModel) maxScroll() int {
  return len(m.blocks) - m.numBlocksShown()
}

func (m *tuiModel) scrollBy(delta int) {
  *m.scroll = util.Clamp(
    *m.scroll + delta,
    0, m.maxScroll(),
  )
}

func (m *tuiModel) moveCursor(delta int) {
  *m.cursor = util.Clamp(
    *m.cursor + delta,
    0, len(m.blocks) - 1,
  )

  if *m.cursor < *m.scroll || *m.cursor >= *m.scroll + m.numBlocksShown() {
    m.scrollBy(delta)
  }
}

const tab = "  "
const defaultNumEntries = 5

func (m tuiModel) View() string {
	s := fmt.Sprintf(
		"%s%s %s",
		ansi.Color("run", "red+b"),
		ansi.Color("me", "white+b"),
		ansi.Color(m.version, "white+d"),
	)

	s += "\n\n"

  for i := *m.scroll; i < *m.scroll + m.numBlocksShown(); i++ {
    block := m.blocks[i]

		active := i == *m.cursor
		_, expanded := m.expanded[i]

		line := " "
		if active {
			line = ">"
		}

		line += " "

		{
			name := block.Name()
			lang := ansi.Color(block.Language(), "white+d")

			if active {
				name = ansi.Color(name, "white+b")
			} else {
				lang = ""
			}

			identifier := fmt.Sprintf(
				"%s %s",
				name,
				lang,
			)

			line += identifier + "\n"
		}

		codeLines := block.Lines()

		for i, codeLine := range codeLines {
			content := tab + tab + codeLine

			if !expanded && len(codeLines) > 1 {
				content += " (...)"
			}

			content = ansi.Color(content, "white+d")

			if i >= 1 && !expanded {
				break
			}

			line += content + "\n"
		}

		s += line
	}

	s += "\n"

	{
		help := strings.Join(
			[]string{
        fmt.Sprintf("%v/%v", *m.cursor + 1, len(m.blocks)),
				"Choose ↑↓←→",
				"Run [Enter]",
				"Expand [Space]",
				"Quit [^C]",
				"  by Stateful",
			},
			tab,
		)

		help = ansi.Color(help, "white+d")

		s += help
	}

	return s
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, isKeyPress := msg.(tea.KeyMsg)

	if isKeyPress {
		switch keyMsg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
      m.moveCursor(-1)

		case "down", "j":
      m.moveCursor(1)

		case " ":
			if _, ok := m.expanded[*m.cursor]; ok {
				delete(m.expanded, *m.cursor)
			} else {
				m.expanded[*m.cursor] = struct{}{}
			}

		case "enter", "l":
			*m.run = m.blocks[*m.cursor]

			return m, tea.Quit
		}
	}

	return m, nil
}

func tuiCmd(
  exitAfterRun *bool,
  numEntries *int,
) *cobra.Command {
	cmd := cobra.Command{
		Use:   "tui",
		Short: "Run the interactive TUI",
		Long:  "Run a command from a descriptive list given by an interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			blocks, err := getCodeBlocks()
			if err != nil {
				return err
			}

			version := "???"

			bi, ok := debug.ReadBuildInfo()
			if ok {
				version = bi.Main.Version
			}

			cursor := 0
			scroll := 0
      isInitialRun := false

			for {
				block := (*document.CodeBlock)(nil)

        if *numEntries <= 0 {
          *numEntries = math.MaxInt32
        }

				model := tuiModel{
					blocks:   blocks,
					version:  version,
					expanded: make(map[int]struct{}),
					run:      &block,
					cursor:   &cursor,
					scroll:   &scroll,
          numEntries: *numEntries,
				}

        if !isInitialRun {
          _, err := fmt.Print("\n")

          if err != nil {
            return err
          }
        }

        isInitialRun = false

				prog := tea.NewProgram(model)
				if _, err := prog.Run(); err != nil {
					return err
				}

        err := error(nil)

				if block != nil {
					err = runBlockCmd(block, cmd, nil)
				} else {
					break
				}

        if err != nil {
          _, err := fmt.Printf(ansi.Color("%v", "red") + "\n", err)

          if err != nil {
            return err
          }
        }

				if cursor < len(blocks)-1 {
					cursor++
				}

				if *exitAfterRun {
					break
				}
			}

			return nil
		},
	}

	setDefaultFlags(&cmd)

  registerTuiCmdFlags(&cmd, exitAfterRun, numEntries)

	return &cmd
}

func registerTuiCmdFlags(
  cmd *cobra.Command,
  exitAfterRun *bool,
  numEntries *int,
) {
	cmd.Flags().BoolVar(exitAfterRun, "exit", false, "Exit runme TUI after running a command")
  cmd.Flags().IntVar(numEntries, "entries", defaultNumEntries, "Number of entries to show in TUI")
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}
