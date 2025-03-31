# Alfred Podcasts Workflow

![icon](icon.png)

A personal project for managing podcasts in Alfred through [Pocket Casts](https://pocketcasts.com/) API.

## Features

### Media Player Integration

I use [IINA](https://github.com/iina/iina) to play podcasts, which supports IPC control via a socket.

The workflow can export the upcoming episode to a playlist for IINA, and sync the playback status back to Pocket Casts.

You can also use Pocket Casts' web player to play your podcasts.

### Usage

- `pc` to list all podcasts
- `pcl` to list latest episodes
- `pcq` to list upcoming episodes (queue)
- `pcs` to search for podcasts for subscribing and unsubscribing

## Installation

Run `make` to compile.
