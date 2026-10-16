# pokedex-cli

A Pokédex you use from the terminal. Walk through location areas in the Pokémon
world, see which Pokémon live there, try to catch them, and look at the stats of
the ones you caught.

All data comes from [PokéAPI](https://pokeapi.co). Responses are cached in
memory, so going back to a page you already visited is instant.

## Requirements

- Go 1.27 or newer
- An internet connection (to reach PokéAPI)

## Getting started

```sh
git clone https://github.com/anggerpputro/pokedex-cli.git
cd pokedex-cli
go build
./pokedex-cli
```

You can also skip the build step and run `go run .` instead.

You'll see a `Pokedex >` prompt. Type a command and press Enter. Commands are
not case-sensitive.

## Commands

| Command             | What it does                                                   |
| ------------------- | -------------------------------------------------------------- |
| `map`               | Show the next 20 location areas                                |
| `mapb`              | Show the previous 20 location areas                            |
| `explore <area>`    | List the Pokémon you can find in an area                       |
| `catch <pokemon>`   | Throw a Pokéball at a Pokémon. It might escape                 |
| `inspect <pokemon>` | Show the height, weight, stats, and types of a caught Pokémon  |
| `pokedex`           | List every Pokémon you've caught                               |
| `help`              | Show all commands                                              |
| `exit`              | Quit the program                                               |

Your Pokédex lives in memory only. Everything you caught is gone when you exit.

## Example session

```text
Pokedex > map
canalave-city-area
eterna-city-area
pastoria-city-area
...
mt-coronet-1f-from-exterior
Pokedex > explore canalave-city-area
Exploring canalave-city-area...
Found Pokemon:
 - tentacool
 - tentacruel
 - staryu
 - magikarp
 ...
Pokedex > catch pidgey
Throwing a Pokeball at pidgey...
[CATCH] catch roll: 5
[CATCH] pokemon base experience: 50
[CATCH] min rate: 50.00
[CATCH] catch chance rate: 10.00
pidgey escaped!
Pokedex > catch pidgey
Throwing a Pokeball at pidgey...
[CATCH] catch roll: 34
[CATCH] pokemon base experience: 50
[CATCH] min rate: 50.00
[CATCH] catch chance rate: 68.00
pidgey was caught!
You may now inspect it with the inspect command.
Pokedex > inspect pidgey
Name: pidgey
Height: 3
Weight: 18
Stats:
  -hp: 40
  -attack: 45
  -defense: 40
  -special-attack: 35
  -special-defense: 35
  -speed: 56
Types:
  - normal
  - flying
Pokedex > pokedex
Your Pokedex:
 - pidgey
Pokedex > exit
Closing the Pokedex... Goodbye!
```

## How it works

### Catching

Look at the two `catch pidgey` attempts above. Pidgey has a base experience of
50. Each throw rolls a random number from 0 to 49:

- First throw: roll 5. That's 5 / 50 = 10%, below the 50% needed, so Pidgey escaped.
- Second throw: roll 34. That's 34 / 50 = 68%, above 50%, so Pidgey was caught.

The rate you need depends on the Pokémon's base experience. Stronger Pokémon
need a higher roll:

| Base experience | Rate needed | Rough chance to catch |
| --------------- | ----------- | --------------------- |
| below 50        | 25%         | 75%                   |
| 50 to 99        | 50%         | 50%                   |
| 100 to 149      | 75%         | 25%                   |
| 150 or more     | 90%         | 10%                   |

The `[CATCH]` lines in the output show each of these numbers, so you can follow
along. The logic is in `commandCatch` in `internal/command.go`.

### Caching

Every PokéAPI response is stored in `PokeCache` (`internal/pokecache.go`). The
cache key is the full request URL. Run `map`, then `mapb`, then `map` again: the
third call doesn't touch the network.

- Each entry expires 5 minutes after it's added.
- `Get` treats an expired entry as missing, even if it's still in the map.
- A background `reapLoop` deletes expired entries every 5 seconds.

## Project structure

```text
.
├── main.go                 # REPL loop: reads input and runs the matching command
├── helper.go               # CleanInput: lowercases input and splits it into words
├── helper_test.go
└── internal/
    ├── command.go          # Every command and its callback
    ├── config.go           # Config: next/previous page URLs for map and mapb
    ├── pokeapi.go          # PokeApi: HTTP client for PokéAPI, goes through the cache
    ├── pokecache.go        # PokeCache: in-memory cache with expiry and reapLoop
    ├── pokecache_test.go
    ├── pokeball.go         # PokeBall: holds the Pokémon you caught
    └── model/              # Structs for PokéAPI JSON responses
        ├── location_area.go
        ├── location_area_encounters.go
        └── pokemon.go
```

## Running tests

```sh
go test ./...
```

The tests cover `CleanInput` and `PokeCache`: adding and reading entries,
expiry, the reap loop, and concurrent access.
