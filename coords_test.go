package cartego

import (
  "math"
  "sort"
  "testing"
)

type floatTest struct {
  start, expected, tolerance float64
}

func (t floatTest) passes(val float64) bool {
  return math.Abs(t.expected-val) <= t.tolerance
}

func TestToRad(t *testing.T) {
  tolerance := .0001
  tests := []floatTest{
    floatTest{0, 0, tolerance},
    floatTest{90, math.Pi/2, tolerance},
    floatTest{111, 1.93731547, tolerance},
    floatTest{-111, -1.93731547, tolerance},
  }

  for _, test := range tests {
    val := toRad(test.start)
    if !test.passes(val) {
      t.Errorf("given: %f; expected: %f; actual: %f", test.start, test.expected, val)
    }
  }
}

func TestToDeg(t *testing.T) {
  tolerance := .0001
  tests := []floatTest{
    floatTest{0, 0, tolerance},
    floatTest{math.Pi/2, 90, tolerance},
    floatTest{1.93731547, 111, tolerance},
    floatTest{-1.93731547, -111, tolerance},
  }

  for _, test := range tests {
    val := toDeg(test.start)
    if !test.passes(val) {
      t.Errorf("given: %f; expected: %f; actual: %f", test.start, test.expected, val)
    }
  }
}

type latTest struct {
  lat float64
  zoom, y, tolerance int
}

func (t latTest) passes(pix int) bool {
  return pix >= t.y && pix - t.y <= t.tolerance
}

func TestLatToYPixels(t *testing.T) {
  tolerance := TILESIZE-1
  tests := []latTest{
    latTest{toRad(40.306107), 17, 49475*TILESIZE, tolerance},
    latTest{toRad(40.306107), 18, 98950*TILESIZE, tolerance},
  }

  for _, test := range tests {
    pix := latToYPixels(test.lat, test.zoom)
    if !test.passes(pix) {
      t.Errorf("given: %f,%d; expected: %d; actual: %d", test.lat, test.zoom, test.y, pix)
    }
  }
}

type lonTest struct {
  lon float64
  zoom, x, tolerance int
}

func (t lonTest) passes(pix int) bool {
  return pix >= t.x && pix - t.x <= t.tolerance
}

func TestLonToXPixels(t *testing.T) {
  tolerance := TILESIZE-1
  tests := []lonTest{
    lonTest{toRad(-111.654995), 17, 24883*TILESIZE, tolerance},
    lonTest{toRad(-111.654995), 18, 49767*TILESIZE, tolerance},
  }

  for _, test := range tests {
    pix := lonToXPixels(test.lon, test.zoom)
    if !test.passes(pix) {
      t.Errorf("given: %f,%d; expected: %d; actual: %d", test.lon, test.zoom, test.x, pix)
    }
  }
}

type mercatorTest struct {
  p Point
  zoom int
  expected Tile
}

func (t mercatorTest) passes(tile Tile) bool {
  return t.expected.X == tile.X && t.expected.Y == tile.Y && t.expected.Zoom == tile.Zoom
}

func TestGetMercatorFromGPS(t *testing.T) {
  tests := []mercatorTest{
    mercatorTest{Point{40.306107, -111.654995}, 17, Tile{X: 24883, Y: 49475, Zoom: 17}},
    mercatorTest{Point{40.306107, -111.654995}, 18, Tile{X: 49767, Y: 98950, Zoom: 18}},
  }

  for _, test := range tests {
    tile := getMercatorFromGPS(test.p, test.zoom)
    if !test.passes(tile) {
      t.Errorf("given: %f,%f,%d; expected: %#v; actual: %#v", test.p.Lat, test.p.Lon, test.zoom, test.expected, tile)
    }
  }
}

type translateTest struct {
  lat, lon, distance, bearing, tolerance float64
  expected Point
}

func (t translateTest) passes(p Point) bool {
  return math.Abs(t.expected.Lat-p.Lat) <= t.tolerance && math.Abs(t.expected.Lon-p.Lon) <= t.tolerance
}

func TestTranslate(t *testing.T) {
  tolerance := 0.001
  tests := []translateTest{
    translateTest{40.306107, -111.654995, 15000, 37, tolerance, Point{40.413889, -111.548333}},
    translateTest{35.696111, 51.423056, 1300, -17, tolerance, Point{35.707222, 51.418889}},
  }

  for _, test := range tests {
    p := translate(test.lat, test.lon, test.distance, test.bearing)
    if !test.passes(p) {
      t.Errorf("given: %f,%f,%f,%f; expected: %#v; actual: %#v", test.lat, test.lon, test.distance, test.bearing, test.expected, p)
    }
  }
}

type tileSorter struct {
  tiles []Tile
}

func (t tileSorter) Len() int {
  return len(t.tiles)
}

func (t tileSorter) Less(i, j int) bool {
  ti, tj := t.tiles[i], t.tiles[j]
  if ti.Zoom == tj.Zoom {
    if ti.Y == tj.Y {
      return ti.X < tj.X
    }
    return ti.Y < tj.Y
  }
  return ti.Zoom < tj.Zoom
}

func (t tileSorter) Swap(i, j int) {
  t.tiles[i], t.tiles[j] = t.tiles[j], t.tiles[i]
}

type getTilesTest struct {
  lat, lon, radius float64
  minZoom, maxZoom int
  expected []Tile
}

func (t getTilesTest) passes(tiles []Tile) bool {
  if len(t.expected) != len(tiles) {
    return false
  }

  exp := tileSorter{t.expected}
  act := tileSorter{tiles}

  sort.Sort(exp)
  sort.Sort(act)

  for i := 0; i < len(tiles); i++ {
    e, a := exp.tiles[i], act.tiles[i]
    if e.X != a.X || e.Y != a.Y || e.Zoom != a.Zoom {
      return false
    }
  }

  return true
}

func TestGetTileCoords(t *testing.T) {
  tests := []getTilesTest{
    getTilesTest{40.306107, -111.654995, .001, 17, 18, []Tile{
        Tile{X: 24883, Y: 49475, Zoom: 17},
        Tile{X: 49767, Y: 98950, Zoom: 18},
      },
    },
  }

  for _, test := range tests {
    tiles := GetTileCoords(test.lat, test.lon, test.radius, test.minZoom, test.maxZoom)
    if !test.passes(tiles) {
      t.Errorf("given: %f,%f,%f,%d,%d; expected: %#v; actual: %#v", test.lat, test.lon, test.radius, test.minZoom, test.maxZoom)
    }
  }
}
