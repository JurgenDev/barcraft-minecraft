package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bwmarrin/discordgo"
	mcquery "github.com/spencersharkey/gomc/query"
)

type Config struct {
	DiscordToken           string `json:"DiscordToken"`
	AWSMachineId           string `json:"AWSMachineId"`
	AWSRegion              string `json:"AWSRegion"`
	MinecraftServerAddress string `json:"MinecraftServerAddress"`
	DiscordHomeChannel     string `json:"DiscordHomeChannel"`
	MinutesToWarning       int    `json:"MinutesToWarning"`
	MinutesToShutdown      int    `json:"MinutesToShutdown"`
}

const (
	SERVER_OFF = iota
	SERVER_ON
)

var config Config
var counter int = 0

func initConfig() error {

	if configfile, openerr := os.Open("config.json"); openerr != nil {
		return fmt.Errorf("Could not find or open config.json")
	} else {
		configbyte, configerr := ioutil.ReadAll(configfile)

		if configerr != nil {
			return fmt.Errorf("No config file found or no access to config file.")
		}

		json.Unmarshal(configbyte, &config)
	}
	return nil
}

func main() {

	if err := initConfig(); err != nil {
		fmt.Println(err)
	}

	discord, derr := discordgo.New("Bot " + config.DiscordToken)
	if derr != nil {
		fmt.Println("Error creating Discord session: ", derr)
		return
	}

	discord.AddHandler(handle_message)

	if openerr := discord.Open(); openerr != nil {
		fmt.Println("Error opening discord connection", openerr)
	}

	defer discord.Close()

	ticker := time.NewTicker(30 * time.Second)
	for _ = range ticker.C {
		check_server_off(discord)
	}

}

func check_server_off(s *discordgo.Session) {
	mcreq := mcquery.NewRequest()

	if mcerr := mcreq.Connect(config.MinecraftServerAddress); mcerr != nil {
		counter = 0
		return
	}

	if res, mcerr := mcreq.Simple(); mcerr != nil {
		counter = 0
		return
	} else {
		if res.NumPlayers == 0 {
			counter++
			switch {
			case counter == 2*config.MinutesToWarning:
				var format = "Server has been empty for %d minutes, stopping server in %d minutes."
				s.ChannelMessageSend(config.DiscordHomeChannel, fmt.Sprintf(format, config.MinutesToWarning, config.MinutesToShutdown-config.MinutesToWarning))
			case counter == 2*config.MinutesToShutdown:
				var format = "Server has been empty for %d minutes, stopping server!"
				s.ChannelMessageSend(config.DiscordHomeChannel, fmt.Sprintf(format, config.MinutesToShutdown))
				server_operation(SERVER_OFF)
			}
		} else {
			counter = 0
		}
	}
}

func handle_message(s *discordgo.Session, m *discordgo.MessageCreate) {
	switch {
	case strings.HasPrefix(m.Content, "!start"):
		s.ChannelMessageSend(m.ChannelID, "Starting BarCraft-Minecraft server! :sunny: Should be joinable in ~15 seconds.")
		server_operation(SERVER_ON)
	case strings.HasPrefix(m.Content, "!stop"):
		s.ChannelMessageSend(m.ChannelID, "Stopping BarCraft-Minecraft server! :crescent_moon:")
		server_operation(SERVER_OFF)
	case strings.HasPrefix(m.Content, "!info"):
		info(s)
	case strings.HasPrefix(m.Content, "!ni"):
		s.ChannelMessageSend(m.ChannelID, "NI")
	}
}

func server_operation(operation int) {

	if sess, awserr := session.NewSession(&aws.Config{Region: aws.String(config.AWSRegion)}); awserr != nil {
		fmt.Print(awserr)
	} else {
		ec2sv := ec2.New(sess)

		var operr error
		switch {
		case operation == SERVER_ON:
			_, operr = ec2sv.StartInstances(&ec2.StartInstancesInput{
				InstanceIds: []*string{
					aws.String(config.AWSMachineId),
				},
			})
		case operation == SERVER_OFF:
			_, operr = ec2sv.StopInstances(&ec2.StopInstancesInput{
				InstanceIds: []*string{
					aws.String(config.AWSMachineId),
				},
			})
		}

		if operr == nil {
			fmt.Print("Succes!")
		} else {
			fmt.Printf("Fails: %s", operr)
		}
	}
}

func info(s *discordgo.Session) {

	var mcreq = mcquery.NewRequest()

	if mcerr := mcreq.Connect(config.MinecraftServerAddress); mcerr != nil {
		s.ChannelMessageSend(config.DiscordHomeChannel, "Server does not appear to be up type !start to start the server!")
	}

	if res, mcerr := mcreq.Simple(); mcerr != nil {
		s.ChannelMessageSend(config.DiscordHomeChannel, "Could not get server info :(")
	} else {
		var players = "No people playing :("
		if res.NumPlayers == 1 {
			players = "One person playing"
		} else if res.NumPlayers > 1 {
			players = fmt.Sprintf("%d player playing right now", res.NumPlayers)
		}

		s.ChannelMessageSend(config.DiscordHomeChannel, fmt.Sprintf("\n Server is running, %s", players))
	}
}
