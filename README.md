# det-blir-nog-regn

<div align="center">
  <img src="https://media4.giphy.com/media/v1.Y2lkPTc5MGI3NjExd3dyMzZlc2p6cmpxam1qNWJoMXRxdnBuYmdiZTc1aWQ5anhqenB0NyZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/2vqaiPr1TrevmxCPUV/giphy.gif" />
</div>

_Because of course it's going to rain. This is Sweden._

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
