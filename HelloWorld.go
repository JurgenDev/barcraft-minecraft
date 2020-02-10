package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bwmarrin/discordgo"
)

func main() {

	token := ""

	discord, derr := discordgo.New("Bot " + token)

	if derr != nil {
		fmt.Println("Error creating Discord session: ", derr)
		return
	}

	discord.AddHandler(handle_message)

	openerr := discord.Open()
	if openerr != nil {
		fmt.Println("Error opening discord connection", openerr)
	}

	fmt.Println("BarCraft is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	discord.Close()
}

func handle_message(s *discordgo.Session, m *discordgo.MessageCreate) {

	if strings.HasPrefix(m.Content, "!start") {
		s.ChannelMessageSend(m.ChannelID, "Starting BarCraft-Minecraft server! Should be joinable in ~30 seconds.")
		start_server()
	}
	if strings.HasPrefix(m.Content, "!stop") {
		s.ChannelMessageSend(m.ChannelID, "Stopping BarCraft-Minecraft server!")
		stop_server()
	}

}

func start_server() {

	sess, awserr := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1"),
	})

	if awserr != nil {
		fmt.Print(awserr)
	}

	ec2sv := ec2.New(sess)

	fmt.Print(ec2sv.DescribeInstances(nil))

	_, starterr := ec2sv.StartInstances(&ec2.StartInstancesInput{
		InstanceIds: []*string{
			aws.String(""),
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
		Region: aws.String("eu-west-1"),
	})

	if awserr != nil {
		fmt.Print(awserr)
	}

	ec2sv := ec2.New(sess)

	fmt.Print(ec2sv.DescribeInstances(nil))

	_, stoperr := ec2sv.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(""),
		},
	})

	if stoperr == nil {
		fmt.Print("Succes!")
	} else {
		fmt.Printf("Fails: %s", stoperr)
	}
}
