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
	MinutesToWarning       string `json:"MinutesToWarning"`
	MinutesToShutdown      string `json:"MinutesToShutdown"`
}

var config Config
var discord discordgo.Session
var counter int = 0

func initConfig() {
	var configfile, _ = os.Open("config.json")
	configbyte, configerr := ioutil.ReadAll(configfile)

	if configerr != nil {
		fmt.Println("No config file found or no access to config file.")
	}

	json.Unmarshal(configbyte, &config)
}

func main() {

	initConfig()

	discord, derr := discordgo.New("Bot " + config.DiscordToken)

	if derr != nil {
		fmt.Println("Error creating Discord session: ", derr)
		return
	}

	discord.AddHandler(handle_message)

	openerr := discord.Open()

	if openerr != nil {
		fmt.Println("Error opening discord connection", openerr)
	}

	ticker := time.NewTicker(30 * time.Second)

	for _ = range ticker.C {
		check_server_off(discord)
	}

	discord.Close()
}

func check_server_off(s *discordgo.Session) {
	var mcreq = mcquery.NewRequest()
	var mcerr = mcreq.Connect(config.MinecraftServerAddress)
	if mcerr != nil {
		counter = 0
		return
	}
	res, mcerr := mcreq.Simple()
	if mcerr != nil {
		counter = 0
		return
	}
	if res.NumPlayers == 0 {
		counter++
		if counter == 2*config.MinutesToWarning {
			s.ChannelMessageSend(config.DiscordHomeChannel, fmt.Sprintf("Server has been empty for %n minutes, stopping server in %n minutes.", config.MinutesToWarning, config.MinutesToShutdown-config.MinutesToWarning))

		} else if counter == 2*config.MinutesToShutdown {
			s.ChannelMessageSend(config.DiscordHomeChannel, fmt.Sprintf("Server has been empty for %n minutes, stopping server!", config.MinutesToShutdown))
			stop_server()
		}
	} else {
		counter = 0
	}
}

func handle_message(s *discordgo.Session, m *discordgo.MessageCreate) {

	if strings.HasPrefix(m.Content, "!start") {
		s.ChannelMessageSend(m.ChannelID, "Starting BarCraft-Minecraft server! :sunny: Should be joinable in ~15 seconds.")
		start_server()
	}
	if strings.HasPrefix(m.Content, "!stop") {
		s.ChannelMessageSend(m.ChannelID, "Stopping BarCraft-Minecraft server! :crescent_moon:")
		stop_server()
	}
	if strings.HasPrefix(m.Content, "!info") {
		info()
	}

}

func start_server() {

	sess, awserr := session.NewSession(&aws.Config{
		Region: aws.String(config.AWSRegion),
	})

	if awserr != nil {
		fmt.Print(awserr)
	}

	ec2sv := ec2.New(sess)

	_, starterr := ec2sv.StartInstances(&ec2.StartInstancesInput{
		InstanceIds: []*string{
			aws.String(config.AWSMachineId),
		},
	})

	if starterr == nil {
		fmt.Print("Succes!")
	} else {
		fmt.Printf("Fails: %s", starterr)
	}
}

func stop_server() {
	sess, awserr := session.NewSession(&aws.Config{
		Region: aws.String(config.AWSRegion),
	})

	if awserr != nil {
		fmt.Print(awserr)
	}

	ec2sv := ec2.New(sess)

	_, stoperr := ec2sv.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(config.AWSMachineId),
		},
	})

	if stoperr == nil {
		fmt.Print("Succes!")
	} else {
		fmt.Printf("Fails: %s", stoperr)
	}
}

func info() {
	var mcreq = mcquery.NewRequest()
	var mcerr = mcreq.Connect(config.MinecraftServerAddress)
	if mcerr != nil {
		discord.ChannelMessageSend(config.DiscordHomeChannel, "Server does not appear to be up type !start to start the server!")
	}
	res, mcerr := mcreq.Simple()

	if mcerr != nil {
		discord.ChannelMessageSend(config.DiscordHomeChannel, "Could not get server info :(")
	} else {
		var players = "er spelen momenteel geen spelers"
		if res.NumPlayers == 1 {
			players = "er speelt momenteel een speler"
		} else if res.NumPlayers > 1 {
			players = fmt.Sprintf("er spelen momenteel %d spelers", res.NumPlayers)
		}

		discord.ChannelMessageSend(config.DiscordHomeChannel, fmt.Sprintf("\n De server draait, %s", players))
	}
}
