package events

import (
	"bufio"
	"fmt"
	"os"
	"proj/handlers"
	"strings"
)

// predeclared variables
var (
	userInputTheme string = "-> "
	userInput      []string
	isGreeting     bool
)

// main loop for each event
func eventMainLoop() {
	reader := bufio.NewReader(os.Stdin)
	for {
		userInput = []string{}

		if !isGreeting {
			fmt.Printf("%v\n", handlers.Greetings)
			isGreeting = true
		}
		fmt.Print("->")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		input = strings.ToLower(input)

		if input == "" {
			continue
		}

		userInput = strings.Fields(input)
		handlers.CommandHandler(userInput)
	}
}

// in future updates
func userInputHandler() {

}
