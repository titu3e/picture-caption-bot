# Picture Caption Bot

This is a lightweight and fast telegram bot which adds random captions to pictures.

## Usage

#### Clone this repository

```shell script
git clone https://github.com/meownoid/picture-caption-bot.git
```

#### Build executable

```shell script
cd picture-caption-bot
go build
```

#### Change config

Edit `config.yaml` or create a new one. Set `token` to your Telegram bot token and `phrases` to phrases you want to use.
Optionally replace `font` with path to your custom font.

#### Start the bot

```shell script
./picture-caption-bot -config config.yaml
```
