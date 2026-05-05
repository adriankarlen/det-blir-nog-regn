# det-blir-nog-regn

Because of course it's going to rain. This is Sweden.

Scrapes today's weather map image from [SVT Väder](https://www.svt.se/vader/vader-idag) and saves it locally. That's it. No forecasting, no AI, no opinions — just the map.

## Usage

```sh
go run main.go
```

Saves `vader_YYYY-MM-DD.jpg` in the current directory.

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
