package domain

import (
	"fmt"
	"math/rand"
	"strings"
)

var Raw string = `#####################################################
#                                                   #
#                                                   #
#                                                   #
#                                                   #
#               ####                  ####          #
#               #  #                ###             #
#               #  #                #               #
#               ####               ##               #
#                                                   #
#                                                   #
#                                                   #
#                                                   #
#                                                   #
#          #####                                    #
#        ###   #                                    #
#       #                                           #
#       #                         #                 #
#       #                        ##                 #
#                             ####                  #
#                          ###                      #
#                                                   #
#                                               ##  #
#                                             ###   #
#                                            ##     #
#                                             #     #
#                                                   #
#####################################################`

type WorldStore interface {
	GetWorld(id string) *World
}

type Layout [][]byte

func ConvertLayout(layout string) Layout {
	lines := strings.Split(layout, "\n")
	result := make(Layout, len(lines))

	for i, line := range lines {
		result[i] = []byte(line)
	}

	return result
}

type World struct {
	ID     string `json:"id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`

	Layout Layout `json:"layout"`
}

func NewWorld(id string, width, height int, layout Layout) *World {
	return &World{
		ID:     id,
		Width:  width,
		Height: height,
		Layout: layout,
	}
}

func (w *World) findRandomSpawnPosition(occupiedPositions map[string]bool) (int, int, error) {
	maxAttempts := 100
	for i := 0; i < maxAttempts; i++ {
		x := rand.Intn(w.Width)
		y := rand.Intn(w.Height)

		if w.Layout[y][x] != '#' && w.Layout[y][x] != '@' {
			key := fmt.Sprintf("%d,%d", x, y)
			if !occupiedPositions[key] {
				return x, y, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("could not find valid spawn position after %d attempts", maxAttempts)
}

func (w *World) SpawnMob(mobType, name, mobID string, occupiedPositions map[string]bool) (*Mob, error) {
	x, y, err := w.findRandomSpawnPosition(occupiedPositions)
	if err != nil {
		return nil, err
	}

	mob := &Mob{
		ID:      mobID,
		Name:    name,
		Type:    mobType,
		WorldID: w.ID,
		X:       x,
		Y:       y,
		Health:  100,
		Attack:  10,
		Defense: 5,
		Symbol:  'M',
	}

	return mob, nil
}
