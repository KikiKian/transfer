package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	choices  []string
	cursor   int
	selected string
}

func initialModel() model {
	return model{
		choices:  []string{"Send", "Accept"},
		cursor:   0,
		selected: "",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.choices[m.cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "\n  Choose an option:\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		if m.cursor == i {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(fmt.Sprintf("%s %s", cursor, choice))
		} else {
			s += fmt.Sprintf("%s %s", cursor, choice)
		}

		if i < len(m.choices)-1 {
			s += "\n"
		}
	}

	s += "\n\n  (Use arrow keys or j/k to navigate, Enter to select, q to quit)\n"

	return s
}

func ShowMenu() string {
	p := tea.NewProgram(initialModel())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		return ""
	}

	if m, ok := finalModel.(model); ok {
		return m.selected
	}

	return ""
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	}

	if cmd != nil {
		cmd.Start()
	}
}

func startHTTPServer(mode string) {
	http.HandleFunc("/send", handleSend)
	http.HandleFunc("/read", handleRead)
	http.HandleFunc("/progress", handleProgress)
	http.HandleFunc("/localip", handleLocalIP)
	http.HandleFunc("/log", handleLog)

	http.Handle("/", http.FileServer(http.Dir("./web")))

	port := ":3030"
	url := "http://localhost:3030/"

	if mode == "Send" {
		url += "send.html"
	} else if mode == "Accept" {
		url += "accept.html"
	}

	fmt.Println("\nStarting server on http://localhost:3030")
	fmt.Printf("Opening %s mode in browser...\n\n", mode)

	// Open browser in a goroutine so it doesn't block the server
	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser(url)
	}()

	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}
