package main

import (
  "flag"
  "fmt"
  "io"
  "path"
  "os"
  "strconv"
  "strings"
  "cartego"
  "github.com/beatgammit/artichoke"
  "time"
)

var runServer bool
var port int
var host string
var strategy string
var minZoom int
var maxZoom int
var downloadDir string
var pause time.Duration
var batchSize int

type cacheLookupTable map[int]map[int]map[int]bool
var cachedTiles cacheLookupTable = make(map[int]map[int]map[int]bool)

func (t cacheLookupTable) Add(tile cartego.Tile) {
  if t[tile.Zoom] == nil {
    t[tile.Zoom] = make(map[int]map[int]bool)
  }
  if t[tile.Zoom][tile.X] == nil {
    t[tile.Zoom][tile.X] = make(map[int]bool)
  }

  t[tile.Zoom][tile.X][tile.Y] = true
}

func (t cacheLookupTable) Lookup(tile cartego.Tile) bool {
  if m, ok := t[tile.Zoom]; ok {
    if m2, ok := m[tile.X]; ok {
      return m2[tile.Y]
    }
  }
  return false
}

const (
  MIN_ZOOM = 1
  MAX_ZOOM = 23
  CONCURRENT_DOWNLOADS = 10
)

func init() {
  flag.BoolVar(&runServer, "server", false, "run the server (requires no arguments)")
  flag.IntVar(&port, "port", 5000, "port to run server on; only valid if -server set as well")
  flag.StringVar(&host, "host", "localhost", "hostname to bind server to; only valid if -server set as well")
  flag.StringVar(&strategy, "strategy", "OpenStreetMaps", "strategy to use (e.g. OpenStreetMaps, Google, Bing, Yahoo, Nokia)")
  flag.StringVar(&downloadDir, "dir", "tiles", "directory for tiles; absolute or relative to the working directory")

  flag.IntVar(&minZoom, "minZoom", 1, fmt.Sprintf("minimum zoom level (%d-%d)", MIN_ZOOM, MAX_ZOOM))
  flag.IntVar(&maxZoom, "maxZoom", 17, fmt.Sprintf("maximum zoom level (%d-%d)", MIN_ZOOM, MAX_ZOOM))

  flag.DurationVar(&pause, "pause", time.Second, "time between batches")
  flag.IntVar(&batchSize, "batch", CONCURRENT_DOWNLOADS, "maximum number of concurrent downloads in a batch")

  flag.Usage = printUsage
  flag.Parse()

  cartego.BatchSize(batchSize)
  cartego.Pause(pause)
}

func printUsage() {
    fmt.Fprintf(os.Stderr, "Usage:\n\n")
    fmt.Fprintf(os.Stderr, "\t%s [flags...] <lat> <lon> <rad>\n\n", os.Args[0])
    fmt.Fprintf(os.Stderr, "Where:\n\n")
    fmt.Fprintf(os.Stderr, "  lat: latitude in decimal degrees\n")
    fmt.Fprintf(os.Stderr, "  lon: longitude in decimal degrees\n")
    fmt.Fprintf(os.Stderr, "  rad: radius in kilometers\n\n")
    fmt.Fprintf(os.Stderr, "The flags are:\n\n")
    flag.PrintDefaults()
}

func main() {
  if minZoom < MIN_ZOOM || minZoom > MAX_ZOOM {
    fmt.Fprintf(os.Stderr, "minZoom not within acceptable range. Given: %d\n\n", minZoom)

    printUsage()
    return
  }

  if maxZoom < MIN_ZOOM || maxZoom > MAX_ZOOM {
    fmt.Fprintf(os.Stderr, "maxZoom not within acceptable range. Given: %d\n\n", minZoom)

    printUsage()
    return
  }

  if minZoom > maxZoom {
    fmt.Fprintf(os.Stderr, "minZoom cannot be greater than maxZoom. minZoom: %d, maxZoom: %d\n\n", minZoom, maxZoom)

    printUsage()
    return
  }

  if runServer {
    if flag.NArg() > 0 {
      fmt.Fprintf(os.Stderr, "Unexpected arguments to cartego server. Aborting.\n\n")

      printUsage()
      return
    }

    startServer()
    return
  } else if flag.NArg() != 3 {
    fmt.Fprintf(os.Stderr, "Invalid number of arguments. Expected 3, given %d\n\n", flag.NArg())

    printUsage()
    return
  }

  lat, err := strconv.ParseFloat(flag.Arg(0), 64)
  if err != nil {
    fmt.Fprintf(os.Stderr, "Expected latitude as first argument, but found: %s\n\n", flag.Arg(0))

    printUsage()
    return
  }

  lon, err := strconv.ParseFloat(flag.Arg(1), 64)
  if err != nil {
    fmt.Fprintf(os.Stderr, "Expected longitude as second argument, but found: %s\n\n", flag.Arg(1))

    printUsage()
    return
  }

  rad, err := strconv.ParseFloat(flag.Arg(2), 64)
  if err != nil {
    fmt.Fprintf(os.Stderr, "Expected radius as third argument, but found: %s\n\n", flag.Arg(2))

    printUsage()
    return
  }

  fmt.Printf("Latitude:  %g°\nLongitude: %g°\nRadius:    %g km\n", lat, lon, rad)

  download(lat, lon, rad, minZoom, maxZoom)
}

func initOutputDir() error {
  return os.MkdirAll(downloadDir, os.ModeDir | os.ModePerm)
}

func readYTiles(zoom, x int) error {
    xDir, err := os.Open(path.Join(downloadDir, strconv.Itoa(zoom), strconv.Itoa(x)))
    if err != nil {
      return fmt.Errorf("Error reading x tile directory: %d/%d", zoom, x)
    }

    defer xDir.Close()

    ys, err := xDir.Readdirnames(-1)
    if err != nil {
      return fmt.Errorf("Error reading x tile directory: %d/%d", zoom, x)
    }

    for _, name := range ys {
      ext := path.Ext(name)
      y, err := strconv.Atoi(name[:len(name)-len(ext)])
      if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing y tile name. zoom: %d, x: %d, y: %s\n", zoom, x, name)
      } else {
        cachedTiles.Add(cartego.Tile{Zoom: zoom, X: x, Y: y})
      }
    }

    return nil
}

func readZoomLevel(zoom int) error {
    zDir, err := os.Open(path.Join(downloadDir, strconv.Itoa(zoom)))
    if err != nil {
      return fmt.Errorf("Error reading zoom directory: %d", zoom)
    }

    defer zDir.Close()

    xs, err := zDir.Readdir(-1)
    if err != nil {
      return fmt.Errorf("Error reading zoom directory: %d", zoom)
    }

    for _, xfi := range xs {
      if !xfi.IsDir() {
        fmt.Fprintln(os.Stderr, "Non-directory found when looking for x tile directory:", xfi.Name())
        continue
      }
      x, err := strconv.Atoi(xfi.Name())
      if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing x tile name. zoom: %d, x: %s\n", zoom, xfi.Name())
      }

      err = readYTiles(zoom, x)
      if err != nil {
        fmt.Fprintln(os.Stderr, "Error reading y tile directory:", err)
      }
    }

    return nil
}

func loadCache() error {
  f, err := os.Open(downloadDir)
  if err != nil {
    return err
  }
  defer f.Close()

  zooms, err := f.Readdir(-1)
  if err != nil {
    return err
  }

  for _, z := range zooms {
    if !z.IsDir() {
      fmt.Fprintln(os.Stderr, "Non-directory found when looking for zoom directory:", z.Name())
      continue
    }

    zoom, err := strconv.Atoi(z.Name())
    if err != nil {
      fmt.Fprintln(os.Stderr, "Error parsing zoom level from file system:", err)
      continue
    }

    err = readZoomLevel(zoom)
    if err != nil {
      fmt.Fprintln(os.Stderr, "Error reading zoom level tiles:", zoom)
    }
  }

  return nil
}

func loadCacheFlat() error {
  f, err := os.Open(downloadDir)
  if err != nil {
    return err
  }
  defer f.Close()

  tiles, err := f.Readdirnames(-1)
  if err != nil {
    return err
  }

  for _, name := range tiles {
    ext := path.Ext(name)

    parts := strings.Split(name[:len(name)-len(ext)], "-")
    if len(parts) != 3 {
      fmt.Fprintln(os.Stderr, "Unrecognized tile format:", name)
      continue
    }

    zoom, err := strconv.Atoi(parts[0])
    if err != nil {
      fmt.Fprintln(os.Stderr, "Error parsing zoom from tile format:", name)
      continue
    }

    x, err := strconv.Atoi(parts[1])
    if err != nil {
      fmt.Fprintln(os.Stderr, "Error parsing x from tile format:", name)
      continue
    }

    y, err := strconv.Atoi(parts[2])
    if err != nil {
      fmt.Fprintln(os.Stderr, "Error parsing y from tile format:", name)
      continue
    }

    cachedTiles.Add(cartego.Tile{Zoom: zoom, X: x, Y: y})
  }

  return nil
}

func removeDuplicates(tiles []cartego.Tile) (ret []cartego.Tile) {
  for _, t := range tiles {
    if !cachedTiles.Lookup(t) {
      ret = append(ret, t)
    }
  }
  return
}

func save(path string, image *cartego.Image, c chan<- bool) {
  f, err := os.Create(path)
  if err != nil {
    fmt.Fprintln(os.Stderr, "Error writing image to file:", err)
    return
  }

  _, err = io.Copy(f, image.Buf)
  if err != nil {
    fmt.Fprintln(os.Stderr, "Error writing image to file:", err)
  }
  c<-true
}

func download(lat, lon, rad float64, minZoom, maxZoom int) {
  if err := initOutputDir(); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
  }

  tiles := cartego.GetTileCoords(lat, lon, rad * 1000, minZoom, maxZoom)
  err := loadCacheFlat()
  if err != nil {
    fmt.Fprintf(os.Stderr, "Error reading cached tiles, assuming none:", err)
  } else {
    tiles = removeDuplicates(tiles)
  }

  var strat cartego.Strategy
  switch strings.ToLower(strategy) {
  case "google":
    strat = cartego.Google
  case "bing":
    strat = cartego.Bing
  case "yahoo":
    strat = cartego.Yahoo
  case "nokia":
    strat = cartego.Nokia
  default:
    strat = cartego.OpenStreetMaps
  }

  done := make(chan bool, CONCURRENT_DOWNLOADS)
  c := cartego.Download(tiles, strat)
  for image := range c {
    ext := ""

    switch image.Type {
    case "image/png":
      ext = ".png"
    case "image/jpeg":
      ext = ".jpg"
    default:
      fmt.Fprintln(os.Stderr, "Unrecognized format, excluding extension:", image.Type)
    }

    fname := fmt.Sprintf("%d-%d-%d", image.Tile.Zoom, image.Tile.X, image.Tile.Y)+ext
    fpath := path.Join(downloadDir, fname)

    go save(fpath, image, done)
  }

  close(done)

  // wait until all tiles have been saved
  for _ = range done {
  }

  fmt.Println("Done!")
}

func startServer() {
  s := artichoke.New(nil, artichoke.Static("./public"))
  s.Run("localhost", 8080)
}
