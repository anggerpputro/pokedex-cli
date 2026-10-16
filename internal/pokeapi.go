package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/anggerpputro/pokedex-cli/internal/model"
)

const baseURL = "https://pokeapi.co"

type PokeApi struct {
	PokeCache PokeCache
}

func NewPokeApi(pokeCache *PokeCache) *PokeApi {
	return &PokeApi{
		PokeCache: *pokeCache,
	}
}

func (p *PokeApi) getPokeApi[T any](path string) (T, error) {
	var data T
	var body []byte

	fullURL := fmt.Sprintf("%s%s", baseURL, strings.Replace(path, baseURL, "", 1))
	cacheKey := fullURL

	cachedBody, cacheOk := p.PokeCache.Get(cacheKey)
	// fmt.Println("[CACHE] check:", cacheOk)
	if cacheOk {
		body = cachedBody
		// fmt.Println("[CACHE] hit:", cacheKey)
	} else {
		// fmt.Println("fullURL:", fullURL)
		res, err := http.Get(fullURL)
		if err != nil {
			return data, fmt.Errorf("error get data from pokeapi: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return data, fmt.Errorf("unexpected PokeAPI status: %d", res.StatusCode)
		}

		// if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		// 	return data, fmt.Errorf("error decoding pokeapi response: %w", err)
		// }

		freshBody, err := io.ReadAll(res.Body)
		if err != nil {
			return data, fmt.Errorf("error reading body response of pokeapi request: %w", err)
		}

		body = freshBody
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return data, fmt.Errorf("error unmarshaling json: %w", err)
	}

	if !cacheOk {
		p.PokeCache.Add(cacheKey, body, 5*time.Minute)
		// fmt.Println("[CACHE] add:", cacheKey)
	}

	return data, nil
}

func (p *PokeApi) GetLocationArea(path string) (model.LocationArea, error) {
	if path == "" {
		path = "/api/v2/location-area/"
	}
	return p.getPokeApi[model.LocationArea](path)
}

func (p *PokeApi) GetLocationAreaEncounters(areaName string) (model.LocationAreaEncounters, error) {
	return p.getPokeApi[model.LocationAreaEncounters](fmt.Sprintf("/api/v2/location-area/%s", areaName))
}

func (p *PokeApi) GetPokemonData(pokemonName string) (model.Pokemon, error) {
	return p.getPokeApi[model.Pokemon](fmt.Sprintf("/api/v2/pokemon/%s", pokemonName))
}
