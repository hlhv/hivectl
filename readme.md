# hivectl

[![A+ Golang report card.](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat)](https://goreportcard.com/report/github.com/hlhv/hivectl)

Hivectl provides functionality to start, stop, and restart cells as background
processes. It automatically handles pidfile creation. It is primarily designed
for use by init systems. It must be run as root.

In order for this tool to work, each cell must:

- Be an executable located somewhere in root's PATH
- Be named `hlhv-<cell name>`
- Have an associated system user of that same name

## Usage

`hivectl <Command> [-h|--help] [-p|--pidfile "<value>"] [-c|-cell "<value>"]`

### Commands

- `start`: Start a cell
- `stop`: Stop a cell
- `restart`: Restart a cell
- `status`: Display cell status

### Arguments

- `-h|--help`: Print help information
- `-p|--pidfile`: Specify the location the pidfile. Defaults to a file named
   `hlhv-<cell name>.pid` located at `/run`
- `-c|--cell`: The cell to control. Default: `queen`
