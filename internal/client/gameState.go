package client

import "fmt"

func (gs *GameState) copyWorldLayout() [][]rune {
	if len(gs.world.Layout) == 0 {
		return nil
	}

	display := make([][]rune, len(gs.world.Layout))
	for i, row := range gs.world.Layout {
		display[i] = make([]rune, len(row))
		for j, cell := range row {
			display[i][j] = rune(cell)
		}
	}
	return display
}

func (gs *GameState) Render() string {
	if len(gs.world.Layout) == 0 {
		return "Loading world...\n"
	}

	display := gs.copyWorldLayout()

	for _, item := range gs.items {
		if item.Y >= 0 && item.Y < len(display) &&
			item.X >= 0 && item.X < len(display[item.Y]) {
			display[item.Y][item.X] = item.Symbol
		}
	}

	for _, mob := range gs.mobs {
		if mob.Y >= 0 && mob.Y < len(display) &&
			mob.X >= 0 && mob.X < len(display[mob.Y]) {
			display[mob.Y][mob.X] = mob.Symbol
		}
	}

	if gs.player.Y >= 0 && gs.player.Y < len(display) &&
		gs.player.X >= 0 && gs.player.X < len(display[gs.player.Y]) {
		display[gs.player.Y][gs.player.X] = '@'
	} else {
		worldWidth := 0
		if len(display) > 0 {
			worldWidth = len(display[0])
		}
		fmt.Printf("DEBUG: Player not drawn - pos (%d, %d), world size (%d, %d)\n",
			gs.player.X, gs.player.Y, worldWidth, len(display))
	}

	var result string
	for _, row := range display {
		result += string(row) + "\n"
	}

	return result
}
