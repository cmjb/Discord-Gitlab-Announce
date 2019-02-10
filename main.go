package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type GitlabPayload struct {
	Object_Kind string
	User_Name   string
	User_Avatar string
	User_Email  string
	Project     struct {
		Name                string
		Path_With_Namespace string
	}
	Repository struct {
		Name     string
		Homepage string
	}
	Commits []struct {
		Id        string
		Message   string
		Timestamp string
	}
}

var discordGlobal *discordgo.Session

type conf struct {
	Token       string `yaml:"token"`
	GitlabToken string `yaml:"gitlab_token"`
	Channel     string `yaml:"channel"`
}

var config conf

func main() {

	configFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		panic(err)
	}

	discord, err := discordgo.New("Bot " + config.Token)
	if err != nil {
		panic(err)
	}

	discordGlobal = discord

	http.HandleFunc("/kurisu/", handleKurisu)
	log.Fatal(http.ListenAndServe("127.0.0.1:8086", nil))
}

func constructEmbed(s *discordgo.Session, payload GitlabPayload) {

	var embedFields []*discordgo.MessageEmbedField

	i := 0

	for _, v := range payload.Commits {

		embedFields = append(embedFields, &discordgo.MessageEmbedField{
			Name:   "Commit ID: " + v.Id,
			Value:  v.Message + "\n" + v.Timestamp,
			Inline: false,
		})
		i++
	}

	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       0x0000ff,
		Description: payload.Project.Path_With_Namespace + "(" + payload.Repository.Name + ")",
		Fields:      embedFields,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: payload.User_Avatar,
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Title:     payload.User_Name + " <" + payload.User_Email + ">",
	}

	s.ChannelMessageSendEmbed(config.Channel, embed)
}

func sendMessage(s *discordgo.Session, msg string) {
	s.ChannelMessageSend(config.Channel, msg)
}

func handleKurisu(w http.ResponseWriter, r *http.Request) {

	headerToken := r.Header.Get("X-Gitlab-Token")
	json := ""
	if strings.Compare(headerToken, config.GitlabToken) == 0 {
		f, err := os.OpenFile("httplogging.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		jsonBody, _ := ioutil.ReadAll(r.Body)

		var gitlabpayload GitlabPayload

		if err := json.Unmarshal(jsonBody, &gitlabpayload); err != nil {
			panic(err)
		}

		log.Print(gitlabpayload.Object_Kind)
		log.SetOutput(f)
		log.Println(gitlabpayload.Object_Kind)

		constructEmbed(discordGlobal, gitlabpayload)
		json = "{status: OK}"

	} else {
		w.WriteHeader(http.StatusForbidden)
		json = "{status: ACCESS FORBIDDEN}"
	}

	fmt.Fprintf(w, "%s", json)
}
