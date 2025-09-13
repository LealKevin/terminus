package domain

import "strings"

var raw string = `
#####################################################
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
	ID     string
	Width  int
	Height int

	Layout Layout
}

func NewWorld(id string, width, height int, layout Layout) *World {
	return &World{
		ID:     id,
		Width:  width,
		Height: height,
		Layout: layout,
	}
}
