package cartego

import(
  "io"
  "net/http"
  "time"
  "fmt"
)

var pause time.Duration
var batchSize int = 1

type Image struct {
  Buf io.Reader
  Type string
  Err error
  Tile Tile
}

func (i *Image) IsZero() bool {
  return i.Buf == nil
}

type Tile struct {
  X, Y, Zoom int
}

type Strategy interface {
  GetPath(Tile, int) string
}

func download(path string, tile Tile, c chan<- *Image, done chan<- bool) {
  if resp, err := http.Get(path); err != nil {
    c<-&Image{Err: err, Buf: nil, Type: "", Tile: tile}
  } else {
    c<-&Image{resp.Body, resp.Header.Get("Content-Type"), nil, tile}
  }
  done<-true
}

func closeWhenDone(c chan<- *Image, exp int, done chan bool) {
  for i := 0; i < exp; i++ {
    <-done
  }
  close(c)
  close(done)
}

// Download initiates downloads for the tiles provided using the given strategy.
func Download(tiles []Tile, strategy Strategy) <-chan *Image {
  if strategy == nil {
    strategy = OpenStreetMaps
  }

  // we need the second channel so we can close the returned channel
  // this makes working with channels easier because you can use a for .. range
  c := make(chan *Image, len(tiles))
  done := make(chan bool, len(tiles))

  go func() {
    numDone := 0
    num := 0
    for i, t := range tiles {
      go download(strategy.GetPath(t, i), t, c, done)
      num++

      if num == batchSize {
        // we've finished this batch, so wait until it's done
        for {
          <-done

          num--
          numDone++

          if numDone == len(tiles) || num == 0 {
            break
          }
        }

        // if we have more to do, sleep a bit first
        if numDone < len(tiles) {
          fmt.Printf("Batch processed: %d/%d\n", numDone, len(tiles))
          time.Sleep(pause)
        }
      }
    }

    closeWhenDone(c, len(tiles)-numDone, done)
  }()

  return c
}

func BatchSize(size int) {
  batchSize = size
}

func Pause(d time.Duration) {
  pause = d
}
