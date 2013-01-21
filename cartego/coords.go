package main

import (
  "cartego"
  "math"
)

// Radius of the earth in meters
const R = 6378100
const TILESIZE = 256

type Point struct {
  Lat, Lon float64
}

func toRad(deg float64) float64 {
  return deg * math.Pi / 180
}

func toDeg(rad float64) float64 {
  return rad * 180 / math.Pi
}

func latToYPixels(lat float64, zoom int) int {
  latM := math.Atanh(math.Sin(lat))
  pixY := -((latM * TILESIZE * math.Exp(float64(zoom) * math.Log(2))) / (2*math.Pi)) + (math.Exp(float64(zoom)*math.Log(2)) * (TILESIZE/2))

  return int(math.Floor(pixY))
}

func lonToXPixels(lon float64, zoom int) int {
  pixX := ((lon * TILESIZE * math.Exp(float64(zoom) * math.Log(2)))) / (2 * math.Pi) + (math.Exp(float64(zoom) * math.Log(2)) * (TILESIZE / 2))

  return int(math.Floor(pixX))
}

func getMercatorFromGPS(p Point, zoom int) cartego.Tile {
  pixX := lonToXPixels(toRad(p.Lon), zoom)
  pixY := latToYPixels(toRad(p.Lat), zoom)
  maxTile := int(math.Pow(2, float64(zoom)))
  maxPix := maxTile * TILESIZE

  if pixX < 0 {
    pixX += maxPix
  } else if pixX > maxPix {
    pixX -= maxPix
  }

  tileX := int(math.Floor(float64(pixX) / TILESIZE))
  tileY := int(math.Floor(float64(pixY) / TILESIZE))
  if tileX >= maxTile {
    tileX -= maxTile
  }

  return cartego.Tile{X: tileX, Y: tileY, Zoom: zoom}
}

func translate(lat, lon, d, bearing float64) Point {
  lat, lon, bearing = toRad(lat), toRad(lon), toRad(bearing)

  lat2 := math.Asin(math.Sin(lat) * math.Cos(d/R) + math.Cos(lat) * math.Sin(d/R) * math.Cos(bearing))
  lon2 := lon + math.Atan2(math.Sin(bearing) * math.Sin(d/R) * math.Cos(lat), math.Cos(d/R) - math.Sin(lat)*math.Sin(lat2))
  lon2 = math.Mod((lon2 + 3*math.Pi), (2 * math.Pi)) - math.Pi

  return Point{toDeg(lat2), toDeg(lon2)}
}

func getTileCoords(lat, lon, radius float64, minZoom, maxZoom int) (ret []cartego.Tile) {
  north := translate(lat, lon, radius, 0)
  south := translate(lat, lon, radius, 180)
  west := translate(lat, lon, radius, 270)
  east := translate(lat, lon, radius, 90)

  for zoom := minZoom; zoom <= maxZoom; zoom++ {
    y0 := getMercatorFromGPS(north, zoom)
    y1 := getMercatorFromGPS(south, zoom)
    x0 := getMercatorFromGPS(west, zoom)
    x1 := getMercatorFromGPS(east, zoom)

    for i := x0.X; i <= x1.X; i++ {
      for j := y0.Y; j <= y1.Y; j++ {
        ret = append(ret, cartego.Tile{X: i, Y: j, Zoom: zoom})
      }
    }
  }

  return ret
}
