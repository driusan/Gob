package css

import (
	"encoding/hex"
	"errors"
	"image/color"
	"strconv"
	"strings"
)

var (
	NoStyles       = errors.New("No styles to apply")
	Invalid        = errors.New("Invalid CSS unit or value")
	NotImplemented = errors.New("Support not yet implemented")
	InheritValue   = errors.New("Value should be inherited")
)

func ConvertUnitToPx(basis int, cssString string) (int, error) {
	if len(cssString) < 2 {
		return basis, Invalid
	}
	if cssString[len(cssString)-2:] == "px" {
		val, _ := strconv.Atoi(cssString[0 : len(cssString)-2])
		return val, nil

	}
	return basis, NotImplemented
}

func hexToUint8(val string) uint8 {
	if len(val) != 2 {
		panic("Invalid input")
	}
	r, err := hex.DecodeString(val)
	if err != nil {
		panic(err)
	}
	return uint8(r[0])
}
func sHexToUint8(val byte) uint8 {
	switch val {
	case '0': // 0x00
		return 0
	case '1': // 0x11
		return 0x11
	case '2':
		return 0x22
	case '3':
		return 0x33
	case '4':
		return 0x44
	case '5':
		return 0x55
	case '6':
		return 0x66
	case '7':
		return 0x77
	case '8':
		return 0x88
	case '9':
		return 0x99
	case 'a', 'A':
		return 0xAA
	case 'b', 'B':
		return 0xBB
	case 'c', 'C':
		return 0xCC
	case 'd', 'D':
		return 0xDD
	case 'e', 'E':
		return 0xEE
	case 'f', 'F':
		return 0xFF
	}
	panic("Invalid character")
}
func ConvertColorToRGBA(cssString string) (*color.RGBA, error) {
	if cssString[0:3] == "rgb" {
		tuple := cssString[4 : len(cssString)-1]
		pieces := strings.Split(tuple, ",")
		if len(pieces) != 3 {
			return nil, Invalid
		}

		rint, _ := strconv.Atoi(strings.TrimSpace(pieces[0]))
		gint, _ := strconv.Atoi(strings.TrimSpace(pieces[1]))
		bint, _ := strconv.Atoi(strings.TrimSpace(pieces[2]))
		return &color.RGBA{uint8(rint), uint8(gint), uint8(bint), 255}, nil

	}
	if cssString[0] == '#' {
		switch len(cssString) {
		case 7:
			// #RRGGBB
			return &color.RGBA{hexToUint8(cssString[1:3]), hexToUint8(cssString[3:5]), hexToUint8(cssString[5:]), 255}, nil
		case 4:
			// #RGB
			return &color.RGBA{sHexToUint8(cssString[1]), sHexToUint8(cssString[2]), sHexToUint8(cssString[3]), 255}, nil
		}
		return nil, Invalid
	}
	switch cssString {
	case "inherit":
		return nil, InheritValue
	case "transparent":
		return &color.RGBA{0x80, 0, 0, 0}, nil
	case "maroon":
		return &color.RGBA{0x80, 0, 0, 255}, nil
	case "red":
		return &color.RGBA{0xff, 0, 0, 255}, nil
	case "orange":
		return &color.RGBA{0xff, 0xa5, 0, 255}, nil
	case "yellow":
		return &color.RGBA{0xff, 0xff, 0, 255}, nil
	case "olive":
		return &color.RGBA{0x80, 0x80, 0, 255}, nil
	case "purple":
		return &color.RGBA{0x80, 0, 0x80, 255}, nil
	case "fuchsia":
		return &color.RGBA{0xff, 0, 0xff, 255}, nil
	case "white":
		return &color.RGBA{0xff, 0xff, 0xff, 255}, nil
	case "lime":
		return &color.RGBA{0, 0xff, 0, 255}, nil
	case "green":
		return &color.RGBA{0, 0x80, 0, 255}, nil
	case "navy":
		return &color.RGBA{0, 0, 0x80, 255}, nil
	case "blue":
		return &color.RGBA{0, 0, 0xff, 255}, nil
	case "aqua":
		return &color.RGBA{0, 0xff, 0xff, 255}, nil
	case "teal":
		return &color.RGBA{0, 0x80, 0x80, 255}, nil
	case "black":
		return &color.RGBA{0, 0, 0, 255}, nil
	case "silver":
		return &color.RGBA{0xc0, 0xc0, 0xc0, 255}, nil
	case "gray", "grey":
		return &color.RGBA{0x80, 0x80, 0x80, 255}, nil
	}
	return nil, NoStyles
}