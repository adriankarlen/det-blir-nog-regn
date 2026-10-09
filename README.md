# det blir nog regn

<div align="center">
  <img src="./assets/were-all-going-to-die.gif" />
</div>

_Because of course it's going to rain. This is Sweden._

Scrapes today's weather map image from [SVT Väder](https://www.svt.se/vader/vader-idag) and saves it locally, along with a short text summary. That's it. No forecasting, no AI, no opinions — just the map and what SVT says about it.

## Usage

```sh
go run main.go
```

Saves two files in the current directory:

- `vader_YYYY-MM-DD.jpg` — today's weather map
- `vader_YYYY-MM-DD.txt` — the article heading and the paragraph under **I DAG**

The summary is also printed to the terminal:

```text
saved: vader_2026-10-09.jpg
saved: vader_2026-10-09.txt

Höstvädret växlar upp

De sista resterna av lågtrycket hänger kvar längs ostkusten och över Östersjön ...
```

If the summary text can't be found, a warning is printed to stderr and the image is still saved.

### Options

| Flag           | Default | Description                  |
| -------------- | ------- | ---------------------------- |
| `-output-dir`  | `.`     | Directory to save the image  |

```sh
go run main.go -output-dir ~/Pictures/vader
```

## Install

```sh
go install github.com/adriankarlen/det-blir-nog-regn@latest
det-blir-nog-regn -output-dir /wherever
```

## Why

Cron job + wallpaper script. Or just morbid curiosity about today's commute.
