package domain

import "strings"

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

//func GetWorld(id string) *World {
//}
