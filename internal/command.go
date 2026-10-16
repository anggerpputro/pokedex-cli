package internal

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/anggerpputro/pokedex-cli/internal/model"
)

type Command struct {
	PokeApi   PokeApi
	PokeCache PokeCache
	PokeBall  PokeBall
}

type CliCommand struct {
	Name        string
	Description string
	Callback    func(*Config, ...string) error
}

func NewCommand(pokeApi *PokeApi, pokeCache *PokeCache) *Command {
	return &Command{
		PokeApi:   *pokeApi,
		PokeCache: *pokeCache,
		PokeBall:  *NewPokeBall(),
	}
}

func (c *Command) GetAvailableCommands() map[string]CliCommand {
	return c.availableCommands()
}

func (c *Command) availableCommands() map[string]CliCommand {
	return map[string]CliCommand{
		"map": {
			Name:        "map",
			Description: "Display the names of 20 location areas in the Pokemon world",
			Callback:    c.commandMap,
		},
		"mapb": {
			Name:        "mapb",
			Description: "Display the previous names of 20 location areas in the Pokemon world",
			Callback:    c.commandMapb,
		},
		"explore": {
			Name:        "explore",
			Description: "Display the names of all Pokémon found in that area",
			Callback:    c.commandExplore,
		},
		"catch": {
			Name:        "catch",
			Description: "Catch Pokemon to add them to the user's Pokedex",
			Callback:    c.commandCatch,
		},
		"inspect": {
			Name:        "inspect",
			Description: "Inspect the Pokemon already caught",
			Callback:    c.commandInspect,
		},
		"pokedex": {
			Name:        "pokedex",
			Description: "Prints a list of all the names of the Pokemon has caught",
			Callback:    c.commandPokedex,
		},
		"help": {
			Name:        "help",
			Description: "Display a help message",
			Callback:    c.commandHelp,
		},
		"exit": {
			Name:        "exit",
			Description: "Exit the pokedex",
			Callback:    commandExit,
		},
	}
}

func (c *Command) commandMap(config *Config, args ...string) error {
	if _, err := c.callPokeApiLocationArea(config, config.Next); err != nil {
		return err
	}
	return nil
}

func (c *Command) commandMapb(config *Config, args ...string) error {
	if config.Previous == "" {
		fmt.Println("you're on the first page")
		return nil
	}
	if _, err := c.callPokeApiLocationArea(config, config.Previous); err != nil {
		return err
	}
	return nil
}

func (c *Command) commandExplore(config *Config, args ...string) error {
	argsLen := len(args)
	if argsLen == 0 {
		return fmt.Errorf("please input the area name")
	}

	areaName := args[0]
	fmt.Printf("Exploring %s...\n", areaName)

	data, err := c.PokeApi.GetLocationAreaEncounters(areaName)
	if err != nil {
		return fmt.Errorf("error get pokeapi location-area encounters: %w", err)
	}

	fmt.Println("Found Pokemon:")
	for _, p := range data.PokemonEncounters {
		fmt.Printf(" - %s\n", p.Pokemon.Name)
	}

	return nil
}

func (c *Command) commandCatch(config *Config, args ...string) error {
	argsLen := len(args)
	if argsLen == 0 {
		return fmt.Errorf("please input the pokemon name")
	}

	pokemonName := args[0]
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemonName)

	_, pokemonCaught := c.PokeBall.Pokedex[pokemonName]
	if pokemonCaught {
		fmt.Printf("%s already caught!\n", pokemonName)
		return nil
	}

	pokemon, err := c.PokeApi.GetPokemonData(pokemonName)
	if err != nil {
		return fmt.Errorf("error get pokeapi pokemin: %w", err)
	}

	roll := rand.Intn(pokemon.BaseExperience)
	catchChanceRate := float64(roll) / float64(pokemon.BaseExperience) * 100

	minRate := 25.00
	if pokemon.BaseExperience >= 150 {
		minRate = 90.00
	} else if pokemon.BaseExperience >= 100 {
		minRate = 75.00
	} else if pokemon.BaseExperience >= 50 {
		minRate = 50.00
	}

	fmt.Printf("[CATCH] catch roll: %d\n", roll)
	fmt.Printf("[CATCH] pokemon base experience: %d\n", pokemon.BaseExperience)
	fmt.Printf("[CATCH] min rate: %.2f\n", minRate)
	fmt.Printf("[CATCH] catch chance rate: %.2f\n", catchChanceRate)

	if catchChanceRate >= minRate {
		c.PokeBall.Add(pokemon)
		fmt.Printf("%s was caught!\n", pokemonName)
		fmt.Println("You may now inspect it with the inspect command.")
	} else {
		fmt.Printf("%s escaped!\n", pokemonName)
	}

	return nil
}

func (c *Command) commandInspect(config *Config, args ...string) error {
	argsLen := len(args)
	if argsLen == 0 {
		return fmt.Errorf("please input the pokemon name")
	}

	pokemonName := args[0]
	pokemon, pokemonCaught := c.PokeBall.Pokedex[pokemonName]
	if pokemonCaught {
		fmt.Printf("Name: %s\n", pokemon.Name)
		fmt.Printf("Height: %d\n", pokemon.Height)
		fmt.Printf("Weight: %d\n", pokemon.Weight)
		fmt.Printf("Stats:\n")
		for _, s := range pokemon.Stats {
			fmt.Printf("  -%s: %v\n", s.Stat.Name, s.BaseStat)
		}
		fmt.Printf("Types:\n")
		for _, t := range pokemon.Types {
			fmt.Printf("  - %s\n", t.Type.Name)
		}
	} else {
		fmt.Printf("%s is not found in your PokeBall!\n", pokemonName)
	}

	return nil
}

func (c *Command) commandPokedex(config *Config, args ...string) error {
	fmt.Println("Your Pokedex:")
	for _, p := range c.PokeBall.Pokedex {
		fmt.Printf(" - %s\n", p.Name)
	}

	return nil
}

func (c *Command) commandHelp(config *Config, args ...string) error {
	fmt.Println("Welcome to the Pokedex!")
	// fmt.Printf("Args: %+v\n", args)
	fmt.Printf("Usage:\n\n")

	for k, v := range c.availableCommands() {
		fmt.Printf("%s: %s\n", k, v.Description)
	}

	return nil
}

func commandExit(config *Config, args ...string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func (c *Command) callPokeApiLocationArea(config *Config, path string) (model.LocationArea, error) {
	data, err := c.PokeApi.GetLocationArea(path)
	if err != nil {
		return model.LocationArea{}, fmt.Errorf("error get pokeapi location-area: %w", err)
	}
	// fmt.Printf("%+v\n", data)

	config.Next = data.Next
	config.Previous = data.Previous

	for _, v := range data.Results {
		fmt.Println(v.Name)
	}

	return data, nil
}
