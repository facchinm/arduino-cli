package shell

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/arduino/arduino-cli/commands"
	"github.com/arduino/arduino-cli/commands/board"
	prompt "github.com/c-bata/go-prompt"
	cobra "github.com/spf13/cobra"
	"github.com/stromland/cobra-prompt"
)

func Quit() *cobra.Command {
	quitCommand := &cobra.Command{
		Aliases: []string{"q"},
		Use:     "quit",
		Short:   "Quit " + commands.AppName,
		Long:    "Quit " + commands.AppName,
		Run:     runQuitCommand,
	}
	return quitCommand
}

func runQuitCommand(cmd *cobra.Command, args []string) {
	os.Exit(0)
}

type Exit int

func exit(_ *prompt.Buffer) {
	panic(Exit(0))
}

func handleExit() {
	switch v := recover().(type) {
	case nil:
		return
	case Exit:
		os.Exit(int(v))
	default:
		fmt.Println(v)
		fmt.Println(string(debug.Stack()))
	}
}

// Init prepares the cobra root command.
func Run(cmd *cobra.Command) {

	defer handleExit()
	var quit = prompt.KeyBind{
		Key: prompt.ControlX,
		Fn:  exit,
	}

	shell := &cobraprompt.CobraPrompt{
		RootCmd:                cmd,
		DynamicSuggestionsFunc: handleDynamicSuggestions,
		ResetFlagsFlag:         true,
		GoPromptOptions: []prompt.Option{
			prompt.OptionTitle(commands.AppName),
			prompt.OptionPrefix(">"),
			prompt.OptionMaxSuggestion(10),
			prompt.OptionAddKeyBind(quit),
			prompt.OptionOnlyUpdateIfSingleChoice(true),
		},
	}
	cmd.PersistentPreRun(cmd, []string{})
	cmd.AddCommand(Quit())
	shell.Run()
}

var requestingBoardCompletion bool

func handleDynamicSuggestions(annotation string, doc prompt.Document) []prompt.Suggest {

	switch annotation {
	case "getBoardsOrFilename":
		if strings.Contains(doc.CurrentLineBeforeCursor(), "-b") {
			requestingBoardCompletion = true
		}
		match := regexp.MustCompile("\\w+:\\w+:\\w+(:\\w+=\\w+)*? ").Match([]byte(doc.CurrentLineBeforeCursor()))
		if match {
			requestingBoardCompletion = false
		}
		if requestingBoardCompletion == true {
			return GetBoards(strings.Count(doc.CurrentLineBeforeCursor(), ":"))
		} else {
			path := doc.GetWordBeforeCursor()
			lastPathSeparator := strings.LastIndex(path, string(os.PathSeparator))
			if lastPathSeparator < 0 {
				lastPathSeparator = 1
				path = "."
			} else if lastPathSeparator == 0 {
				lastPathSeparator = 1
			}
			return GetFilename(path[:lastPathSeparator])
		}
	default:
		return []prompt.Suggest{}
	}
}

func GetBoards(howSpecified int) []prompt.Suggest {
	suggestions := []prompt.Suggest{}
	list := board.CreateAllKnownBoardsList()

	for _, el := range list.Boards {
		extra := ""
		// Search if it has additional fqbn Properties
		if len(el.Properties) > 0 && howSpecified > 2 {
			for k, _ := range el.Properties {
				split := strings.Split(k, ".")
				if split[0] == "menu" {
					extra = ":" + split[1] + "=" + split[2]
					suggestions = AppendIfMissing(suggestions, prompt.Suggest{el.Fqbn + extra, el.Name, true, true})
				}
			}
		} else {

			split := strings.Split(el.Fqbn, ":")
			if howSpecified == 0 {
				suggestions = AppendIfMissing(suggestions, prompt.Suggest{split[0], "", true, false})
			}
			if howSpecified == 1 {
				suggestions = AppendIfMissing(suggestions, prompt.Suggest{split[0] + ":" + split[1], "", true, false})
			}
			if howSpecified == 2 {
				suggestions = AppendIfMissing(suggestions, prompt.Suggest{el.Fqbn, el.Name, true, false})
			}
		}
	}
	return suggestions
}

func GetFilename(startPath string) []prompt.Suggest {
	suggestions := []prompt.Suggest{}

	// Allow relative paths and filter by extension
	files, err := ioutil.ReadDir(startPath)
	if err != nil {
		return suggestions
	}

	for _, f := range files {
		name := f.Name()
		if f.IsDir() {
			name = startPath + string(os.PathSeparator) + name
		}
		name = filepath.Clean(name)
		if f.IsDir() {
			name = name
			suggestions = append(suggestions, prompt.Suggest{name + string(os.PathSeparator), "", true, false})
		}
		if strings.HasSuffix(f.Name(), ".ino") {
			suggestions = append(suggestions, prompt.Suggest{name, "", true, false})
		}
	}
	return suggestions
}

func AppendIfMissing(slice []prompt.Suggest, i prompt.Suggest) []prompt.Suggest {
	for _, ele := range slice {
		if ele.Text == i.Text {
			return slice
		}
	}
	return append(slice, i)
}
