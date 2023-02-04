package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/davecgh/go-spew/spew"
	"github.com/mdp/qrterminal/v3"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var token = os.Getenv("DISCORD_TOKEN")

var channelID = os.Getenv("DISCORD_CHANNEL_ID")

var whatsappChannelID = os.Getenv("WHATSAPP_CHANNEL_ID")

var (
	discord *discordgo.Session
	client  *whatsmeow.Client
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())

		if v.Info.Chat.String() != whatsappChannelID {
			return
		}

		if v.Message.SenderKeyDistributionMessage != nil {
			return
		}

		spew.Dump(v)

		author := &discordgo.MessageEmbedAuthor{
			Name:    v.Info.PushName,
			IconURL: "http://s3.cri.epita.fr/cri-intranet/img/blank.jpg",
		}

		content := v.Message.GetConversation()
		if v.Info.MediaType == "" {
			if v.RawMessage.ExtendedTextMessage != nil {
				if v.RawMessage.ExtendedTextMessage.Text != nil {
					content = *v.RawMessage.ExtendedTextMessage.Text
				}
			}

			discord.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
				Description: content,
				Timestamp:   v.Info.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
				Author:      author,
				Color:       0x25D366,
			})

			return
		}

		var title string
		var mimeType string
		var caption string
		if v.Info.MediaType == "image" {
			mimeType = *v.RawMessage.ImageMessage.Mimetype
			title = "image." + mimeType[6:]
			if v.RawMessage.ImageMessage.Caption != nil {
				caption = *v.RawMessage.ImageMessage.Caption
			}
		} else if v.Info.MediaType == "document" {
			if v.IsDocumentWithCaption {
				title = *v.RawMessage.DocumentWithCaptionMessage.Message.DocumentMessage.Title
				mimeType = *v.RawMessage.DocumentWithCaptionMessage.Message.DocumentMessage.Mimetype
				caption = *v.RawMessage.DocumentWithCaptionMessage.Message.DocumentMessage.Caption
			} else {
				title = *v.RawMessage.DocumentMessage.Title
				mimeType = *v.RawMessage.DocumentMessage.Mimetype
			}
		} else {
			fmt.Println("Unknown media type:", v.Info.MediaType)
			return
		}

		if v.Info.MediaType == "image" || v.Info.MediaType == "document" {
			fileBytes, err := client.DownloadAny(v.Message)
			if err != nil {
				fmt.Println("Error downloading file:", err)
				return
			}

			if caption != "" {
				discord.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
					Description: caption,
					Timestamp:   v.Info.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
					Author:      author,
					Color:       0x25D366,
				})
			}

			discord.ChannelFileSend(channelID, title, bytes.NewReader(fileBytes))
		}
	}
}

func main() {
	var err error
	discord, err = discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:session.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// list groups
	groups, err := client.GetJoinedGroups()
	if err != nil {
		panic(err)
	}
	for _, group := range groups {
		fmt.Println(group.Name, group.JID)
	}

	err = discord.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
	discord.Close()
}
