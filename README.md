# Whatsapp-Discord Bridge
Forwards messages from a Whatsapp group to a Discord channel.

## Usage
```
docker run -it --rm -e DISCORD_TOKEN=X -e DISCORD_CHANNEL_ID=12345 -e WHATSAPP_CHANNEL_ID=1234@g.us -v $PWD/data:/app aureleoules/ffo
```
