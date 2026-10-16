package internal

import (
	"sync"

	"github.com/anggerpputro/pokedex-cli/internal/model"
)

type PokeBall struct {
	Pokedex map[string]model.Pokemon
	mutex   *sync.RWMutex
}

func NewPokeBall() *PokeBall {
	return &PokeBall{
		Pokedex: make(map[string]model.Pokemon),
		mutex:   &sync.RWMutex{},
	}
}

func (pb *PokeBall) Add(pokemon model.Pokemon) {
	pb.mutex.Lock()
	pb.Pokedex[pokemon.Name] = pokemon
	pb.mutex.Unlock()
}
