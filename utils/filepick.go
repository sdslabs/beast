package utils

import (
	"fmt"
	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/pkg/homedir"
	log "github.com/sirupsen/logrus"
)

var publicKeyFiles = []string{
	".pub",
}

type filePickerModel struct {
	fp     filepicker.Model
	choice string
}

func (model filePickerModel) Init() tea.Cmd {
	return model.fp.Init()
}

func (model filePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch message := msg.(type) {
	case tea.KeyMsg:
		if messageString := message.String(); messageString == "q" || messageString == "ctrl+c" {
			return model, tea.Quit
		}
	}

	model.fp, cmd = model.fp.Update(msg)

	if selected, path := model.fp.DidSelectFile(msg); selected {
		model.choice = path
		return model, tea.Quit
	}

	return model, cmd
}

func (model filePickerModel) View() string {
	if model.choice != "" {
		return fmt.Sprintf("Selected File: %s\n", model.choice)
	}

	return model.fp.View()
}

var filePicker = filepicker.New()

func PromptPublicKeyFile() (string, error) {
	log.Infoln("Choose public key file... press 'q' to quit")

	filePicker.ShowHidden = true
	filePicker.AllowedTypes = publicKeyFiles
	filePicker.CurrentDirectory = homedir.Get()

	model := filePickerModel{
		fp: filePicker,
	}

	final, err := tea.NewProgram(model).Run()
	if err != nil {
		return "", err
	}

	m := final.(filePickerModel)
	return m.choice, nil
}
