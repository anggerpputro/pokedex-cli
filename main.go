package main

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/anggerpputro/pokedex-cli/internal"
)

func main() {
	config := internal.Config{}
	pokeCache := internal.NewPokeCache(5 * time.Second)
	command := internal.NewCommand(internal.NewPokeApi(pokeCache), pokeCache)
	commands := command.GetAvailableCommands()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Pokedex > ")
		if scanner.Scan() {
			userInput := scanner.Text()
			cleanedInput := CleanInput(userInput)
			firstInput := cleanedInput[0]

			command, ok := commands[firstInput]
			if ok {
				config.Commands = append(config.Commands, userInput)
				err := command.Callback(&config, cleanedInput[1:]...)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error calling callback function: %v\n", err)
				}
			} else {
				fmt.Println("Unknown command, type help for available commands")
			}
			// fmt.Println("Your command was:", cleanedInput[0])
		} else {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "error scanning for the input: %v\n", scanner.Err())
			}
			break
		}
	}
}
