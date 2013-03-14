package cartego

import (
	"fmt"
	"strconv"
	"strings"
)

var osmAlphabet []string = []string{"a", "b", "c"}
var googGalileos []string

var OpenStreetMaps Strategy = &openStreetMaps{}
var Google Strategy = &google{}
var Bing Strategy = &bing{}
var Yahoo Strategy = &yahoo{}
var Nokia Strategy = &nokia{}

func init() {
	// the s param for a Google image string is one of:
	// ["G", "Ga", "Gal", ..., "Galileo"]
	galileo := "Galileo"
	for i := 0; i < len(galileo); i++ {
		googGalileos = append(googGalileos, galileo[:i+1])
	}
}

type openStreetMaps struct {
}

func (s *openStreetMaps) GetPath(t Tile, i int) string {
	return fmt.Sprintf("http://%s.tile.openstreetmap.org/%d/%d/%d.png", osmAlphabet[i%3], t.Zoom, t.X, t.Y)
}

type google struct {
	j int
}

func (s *google) GetPath(t Tile, i int) string {
	if i > 0 {
		s.j = i % 2
	} else {
		s.j = s.j % 2
	}

	galileo := googGalileos[s.j%len(googGalileos)]

	path := fmt.Sprintf("http://khm%d.google.com/kh/v=125&x=%d&y=%d&z=%d&s=%s", s.j%2, t.X, t.Y, t.Zoom, galileo)
	return path
}

type bing struct {
}

// really just a convenience function, but it's only used in Bing's strategy
func (s *bing) pad(n string, p int) string {
	if len(n) < p {
		n = strings.Repeat("0", p-len(n)) + n
	}
	return n
}

func (s *bing) GetPath(t Tile, _ int) string {
	quadkey := "0"
	for i := t.Zoom; i > 0; i-- {
		mask := 1 << uint(i-1)
		if t.Y&mask != 0 {
			quadkey += "1"
		} else {
			quadkey += "0"
		}
		if t.X&mask != 0 {
			quadkey += "1"
		} else {
			quadkey += "0"
		}
	}

	key, _ := strconv.ParseInt(quadkey, 2, 64)
	skey := s.pad(strconv.FormatInt(key, 4), t.Zoom)

	return fmt.Sprintf("http://ecn.t3.tiles.virtualearth.net/tiles/a%s.jpeg?g=915&mkt=en-us&n=z", skey)
}

type yahoo struct {
}

func (s *yahoo) GetPath(t Tile, _ int) string {
	return fmt.Sprintf("http://4.maptile.lbs.ovi.com/maptiler/v2/maptile/279af375be/satellite.day/%d/%d/%d/256/jpg?lg=ENG&token=TrLJuXVK62IQk0vuXFzaig%%3D%%3D&requestid=yahoo.prod&app_id=eAdkWGYRoc4RfxVo0Z4B", t.Zoom, t.X, t.Y)
}

type nokia struct {
}

func (s *nokia) GetPath(t Tile, _ int) string {
	return fmt.Sprintf("http://4.maptile.lbs.ovi.com/maptiler/v2/maptile/4176ef2b30/satellite.day/%d/%d/%d/256/png8?token=fee2f2a877fd4a429f17207a57658582&appId=nokiaMaps", t.Zoom, t.X, t.Y)
}
