package cmd

import (
	"fmt"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	"github.com/stateful/runme/internal/document"
)

type TUIModel struct {
  blocks document.CodeBlocks
  cursor int
  expanded map[int]struct{}
  version string
  run **document.CodeBlock
}

const TAB = "  "

func (m TUIModel) View() string {
  s := fmt.Sprintf(
    "%s%s %s",
    ansi.Color("run", "red+b"),
    ansi.Color("me", "white+b"),
    ansi.Color(m.version, "white+d"),
  )

  s += "\n\n"

  for i, block := range m.blocks {
    active := i == m.cursor
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
      content := TAB + TAB + codeLine

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
        "Choose ↑↓←→",
        "Run [Enter]",
        "Expand [Space]",
        "  by Stateful",
      },
      TAB,
    )

    help = ansi.Color(help, "white+d")

    s += help
  }

  return s
}

func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  switch msg := msg.(type) {
  case tea.KeyMsg: 
    switch msg.String() {
    case "ctrl+c", "q":
      return m, tea.Quit
    

    case "up", "k":
      if m.cursor > 0 {
          m.cursor--
      }

    case "down", "j":
      if m.cursor < len(m.blocks)-1 {
          m.cursor++
      }

    case " ":
      if _, ok := m.expanded[m.cursor]; ok {
        delete(m.expanded, m.cursor)
      } else {
        m.expanded[m.cursor] = struct{}{}
      }

    case "enter", "l":
      *m.run = m.blocks[m.cursor]

      return m, tea.Quit
    }
  }

  return m, nil
}

func tuiCmd() *cobra.Command {
  return &cobra.Command {
     Use: "tui",
     Short: "Run the interactive TUI",
     Long: "Run a command from a descriptive list given by an interactive TUI",
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

       block := (*document.CodeBlock)(nil)

       model := TUIModel {
         blocks: blocks,
         version: version,
         expanded: make(map[int]struct{}),
         run: &block,
       }

       prog := tea.NewProgram(model)
       if _, err := prog.Run(); err != nil {
         return err
       }

       if block != nil {
         runBlockCmd(block, cmd, nil)
       } else { }

       return nil
     },
   }
}

func (m TUIModel) Init() tea.Cmd {
  return nil
}
