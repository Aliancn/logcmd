package main

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyMatching(t *testing.T) {
	quit := key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "退出"),
	)

	escMsg := tea.KeyMsg{Type: tea.KeyEsc}

	if key.Matches(escMsg, quit) {
		t.Errorf("Esc matched Quit (ctrl+c)!")
	} else {
		fmt.Println("Esc did not match Quit (ctrl+c). Correct.")
	}
    
    // Check if adding 'q' makes it match
    quitWithQ := key.NewBinding(
        key.WithKeys("ctrl+c", "q"),
    )
    if key.Matches(escMsg, quitWithQ) {
        fmt.Println("Esc matched Quit (ctrl+c, q)? No.")
    } else {
        fmt.Println("Esc did not match Quit (ctrl+c, q). Correct.")
    }
}
